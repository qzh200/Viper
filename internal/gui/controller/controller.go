package controller

import (
	"fmt"
	"sync"
	"time"

	"viper/internal/domain"
	"viper/internal/gui/model"
	"viper/internal/gui/theme"
)

const DefaultRefresh = 2 * time.Second

type Hardware interface {
	Monitor() (domain.MonitorData, error)
	PowerMode() (domain.PowerMode, error)
	SetPowerMode(mode domain.PowerMode) error
	Route() string
	Instance() (string, error)
	FanState() (domain.FanState, error)
	SetFans(config domain.FanConfig) error
	SetCoolBoost(enabled bool) error
	Lighting() (domain.RGBConfig, error)
	SetLighting(config domain.RGBConfig) error
	ResetLighting() error
	BacklightTimeout() (bool, error)
	SetBacklightTimeout(enabled bool) error
	GPUMode() (domain.GPUMode, error)
	SetGPUMode(mode domain.GPUMode) error
	DiscreteGPU() (bool, error)
	LCDOverdrive() (bool, error)
	SetLCDOverdrive(enabled bool) error
	WinKeyLocked() (bool, error)
	SetWinKeyLocked(locked bool) error
	BootSound() (bool, error)
	SetBootSound(enabled bool) error
}

type AppearanceStore interface {
	Theme() string
	SetTheme(mode string) error
}

type action struct {
	run     func() error
	onError func(error)
}

type Controller struct {
	hw    Hardware
	store AppearanceStore

	mu        sync.RWMutex
	snap      model.Overview
	err       string
	history   []model.Sample
	smoother  *model.Smoother
	smoothing bool

	fanTimerMu    sync.Mutex
	fanTimer      *time.Timer
	fanMu         sync.Mutex
	pendingFan    *domain.FanConfig
	pendingFanErr func(error)
	fanInFlight   bool

	lightTimerMu    sync.Mutex
	lightTimer      *time.Timer
	lightMu         sync.Mutex
	pendingLight    *domain.RGBConfig
	pendingLightErr func(error)
	lightInFlight   bool

	actions chan action
	stop    chan struct{}
	once    sync.Once
}

func New(hw Hardware, store AppearanceStore) *Controller {
	c := &Controller{
		hw:       hw,
		store:    store,
		smoother: model.NewSmoother(),
		actions:  make(chan action, 16),
		stop:     make(chan struct{}),
	}
	c.Refresh()
	return c
}

func (c *Controller) Snapshot() model.Overview {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snap
}

func (c *Controller) Refresh() {
	overview, monitor, failure := c.read()

	c.mu.Lock()
	defer c.mu.Unlock()
	if failure != "" {
		c.err = failure
	} else if c.err != "" && overview.Error == "" {
		overview.Error = c.err
	}
	c.snap = overview
	if monitor == nil {
		return
	}

	if c.smoothing {
		c.smoother.SetTarget(*monitor)
	} else {
		c.smoother.Snap(*monitor)
	}
	smoothMon, smoothMetrics := c.smoother.Current()
	c.snap.Monitor = smoothMon
	c.snap.Metrics = smoothMetrics

	c.history = append(c.history, model.Sample{
		Time:       overview.Updated,
		CPUTempC:   monitor.CPUTemperatureC,
		GPUTempC:   monitor.GPUTemperatureC,
		BoardTempC: monitor.SystemTempC,
		CPUUsage:   monitor.CPUUsagePercent,
		GPUUsage:   float64(monitor.GPUUsagePercent),
	})
	if len(c.history) > model.HistoryLimit {
		c.history = c.history[len(c.history)-model.HistoryLimit:]
	}
	c.snap.History = append([]model.Sample(nil), c.history...)
}

func (c *Controller) read() (model.Overview, *domain.MonitorData, string) {
	overview := model.Overview{PowerModes: domain.AvailablePowerModes()}

	instance, err := c.hw.Instance()
	if err != nil {
		overview.Error = err.Error()
		return overview, nil, err.Error()
	}
	overview.Connected = true
	overview.Instance = instance
	overview.Route = c.hw.Route()

	if mode, err := c.hw.PowerMode(); err == nil {
		overview.PowerMode = mode
	} else {
		overview.Error = err.Error()
	}

	if fanState, err := c.hw.FanState(); err == nil {
		overview.FanState = fanState
	}

	if lighting, err := c.hw.Lighting(); err == nil {
		overview.Lighting = lighting
	}

	if timeout, err := c.hw.BacklightTimeout(); err == nil {
		overview.BacklightTimeout = timeout
	}

	if gpuMode, err := c.hw.GPUMode(); err == nil {
		overview.GPUMode = gpuMode
	}

	if discrete, err := c.hw.DiscreteGPU(); err == nil {
		overview.DiscreteGPU = discrete
	}

	if overdrive, err := c.hw.LCDOverdrive(); err == nil {
		overview.LCDOverdrive = overdrive
	}

	if winLocked, err := c.hw.WinKeyLocked(); err == nil {
		overview.WinKeyLocked = winLocked
	}

	if bootSound, err := c.hw.BootSound(); err == nil {
		overview.BootSound = bootSound
	}

	monitor, err := c.hw.Monitor()
	if err != nil {
		overview.Error = err.Error()
		return overview, nil, err.Error()
	}

	overview.Metrics = model.Metrics(monitor)
	overview.Monitor = monitor
	overview.Updated = time.Now()
	return overview, &monitor, ""
}

func (c *Controller) Work() {
	for {
		select {
		case pending := <-c.actions:
			c.executeAction(pending)
			c.Refresh()
		default:
			return
		}
	}
}

func (c *Controller) executeAction(pending action) {
	err := pending.run()
	if pending.onError != nil {
		pending.onError(err)
	}
	c.mu.Lock()
	if err != nil {
		c.err = err.Error()
	} else {
		c.err = ""
	}
	c.mu.Unlock()
}

func (c *Controller) StepSmooth(alpha float64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.smoother.Step(alpha) {
		return false
	}

	smoothMon, smoothMetrics := c.smoother.Current()
	c.snap.Monitor = smoothMon
	c.snap.Metrics = smoothMetrics
	return true
}

func (c *Controller) SetSmoothing(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.smoothing = enabled
}

func (c *Controller) Start(interval time.Duration, onChange func()) {
	c.SetSmoothing(true)
	if interval <= 0 {
		interval = DefaultRefresh
	}
	go func() {
		hwTicker := time.NewTicker(interval)
		smoothTicker := time.NewTicker(25 * time.Millisecond)
		defer hwTicker.Stop()
		defer smoothTicker.Stop()

		for {
			select {
			case <-c.stop:
				return
			case pending := <-c.actions:
				c.executeAction(pending)
				c.Refresh()
				if onChange != nil {
					onChange()
				}
			case <-hwTicker.C:
				c.Work()
				c.Refresh()
				if onChange != nil {
					onChange()
				}
			case <-smoothTicker.C:
				if c.StepSmooth(model.DefaultSmoothAlpha) {
					if onChange != nil {
						onChange()
					}
				}
			}
		}
	}()
}

func (c *Controller) Stop() {
	c.once.Do(func() {
		close(c.stop)
		c.fanTimerMu.Lock()
		if c.fanTimer != nil {
			c.fanTimer.Stop()
		}
		c.fanTimerMu.Unlock()
		c.fanMu.Lock()
		c.pendingFan = nil
		c.fanMu.Unlock()

		c.lightTimerMu.Lock()
		if c.lightTimer != nil {
			c.lightTimer.Stop()
		}
		c.lightTimerMu.Unlock()
		c.lightMu.Lock()
		c.pendingLight = nil
		c.lightMu.Unlock()
	})
}

func (c *Controller) SetPowerMode(mode domain.PowerMode, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetPowerMode(mode) }, onError)
}

func (c *Controller) SetFanConfig(config domain.FanConfig, onError func(error)) bool {
	c.fanMu.Lock()
	c.pendingFan = &config
	c.pendingFanErr = onError
	if c.fanInFlight {
		c.fanMu.Unlock()
		return true
	}
	c.fanInFlight = true
	c.fanMu.Unlock()

	return c.enqueue(func() error {
		for {
			c.fanMu.Lock()
			cfg := c.pendingFan
			errCb := c.pendingFanErr
			c.pendingFan = nil
			c.pendingFanErr = nil
			if cfg == nil {
				c.fanInFlight = false
				c.fanMu.Unlock()
				return nil
			}
			c.fanMu.Unlock()

			err := c.hw.SetFans(*cfg)
			if errCb != nil {
				errCb(err)
			}
			if err != nil {
				c.fanMu.Lock()
				c.fanInFlight = false
				c.fanMu.Unlock()
				return err
			}
		}
	}, onError)
}

func (c *Controller) SetFanConfigDebounced(config domain.FanConfig, delay time.Duration, onError func(error)) {
	c.fanTimerMu.Lock()
	defer c.fanTimerMu.Unlock()

	if delay <= 0 {
		delay = 180 * time.Millisecond
	}

	if c.fanTimer != nil {
		c.fanTimer.Stop()
	}

	c.fanTimer = time.AfterFunc(delay, func() {
		c.SetFanConfig(config, onError)
	})
}

func (c *Controller) SetCoolBoost(enabled bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetCoolBoost(enabled) }, onError)
}

func (c *Controller) SetLighting(config domain.RGBConfig, onError func(error)) bool {
	c.lightMu.Lock()
	c.pendingLight = &config
	c.pendingLightErr = onError
	if c.lightInFlight {
		c.lightMu.Unlock()
		return true
	}
	c.lightInFlight = true
	c.lightMu.Unlock()

	return c.enqueue(func() error {
		for {
			c.lightMu.Lock()
			cfg := c.pendingLight
			errCb := c.pendingLightErr
			c.pendingLight = nil
			c.pendingLightErr = nil
			if cfg == nil {
				c.lightInFlight = false
				c.lightMu.Unlock()
				return nil
			}
			c.lightMu.Unlock()

			err := c.hw.SetLighting(*cfg)
			if errCb != nil {
				errCb(err)
			}
			if err != nil {
				c.lightMu.Lock()
				c.lightInFlight = false
				c.lightMu.Unlock()
				return err
			}
		}
	}, onError)
}

func (c *Controller) SetLightingDebounced(config domain.RGBConfig, delay time.Duration, onError func(error)) {
	c.lightTimerMu.Lock()
	defer c.lightTimerMu.Unlock()

	if delay <= 0 {
		delay = 180 * time.Millisecond
	}

	if c.lightTimer != nil {
		c.lightTimer.Stop()
	}

	c.lightTimer = time.AfterFunc(delay, func() {
		c.SetLighting(config, onError)
	})
}

func (c *Controller) ResetLighting(onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.ResetLighting() }, onError)
}

func (c *Controller) SetBacklightTimeout(enabled bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetBacklightTimeout(enabled) }, onError)
}

func (c *Controller) SetGPUMode(mode domain.GPUMode, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetGPUMode(mode) }, onError)
}

func (c *Controller) SetLCDOverdrive(enabled bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetLCDOverdrive(enabled) }, onError)
}

func (c *Controller) SetWinKeyLocked(locked bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetWinKeyLocked(locked) }, onError)
}

func (c *Controller) SetBootSound(enabled bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetBootSound(enabled) }, onError)
}

func (c *Controller) enqueue(run func() error, onError ...func(error)) bool {
	pending := action{run: run}
	if len(onError) > 0 {
		pending.onError = onError[0]
	}
	select {
	case c.actions <- pending:
		return true
	default:
		return false
	}
}

func (c *Controller) Samples() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.history)
}

func (c *Controller) ThemeMode() theme.Mode {
	if c.store == nil {
		return theme.ModeAuto
	}
	mode, ok := theme.Parse(c.store.Theme())
	if !ok {
		return theme.ModeAuto
	}
	return mode
}

func (c *Controller) SetThemeMode(mode theme.Mode) error {
	if !mode.Valid() {
		return fmt.Errorf("controller: 不认识的外观模式 %q", mode)
	}
	if c.store == nil {
		return nil
	}
	return c.store.SetTheme(string(mode))
}
