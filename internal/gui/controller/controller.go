// Package controller 编排硬件读写与界面状态。
//
// 界面只经过 Controller 访问机器：Controller 依赖的是接口（Hardware、
// AppearanceStore），而不是 internal/app 里的具体实现，因此界面层可以
// 脱离原厂服务单独测试。
package controller

import (
	"fmt"
	"sync"
	"time"

	"viper/internal/domain"
	"viper/internal/gui/model"
	"viper/internal/gui/theme"
)

// DefaultRefresh 是后台刷新快照的默认周期。
//
// 两秒一次：曲线上每两秒一个点，十个点正好是最近二十秒。
const DefaultRefresh = 2 * time.Second

// Hardware 是界面需要的硬件能力。internal/app.App 满足这个接口。
type Hardware interface {
	// Monitor 读取整机遥测。
	Monitor() (domain.MonitorData, error)
	// PowerMode 读取当前生效的性能档位。
	PowerMode() (domain.PowerMode, error)
	// SetPowerMode 切换性能档位。
	SetPowerMode(mode domain.PowerMode) error
	// Route 返回当前连接通道的名字。
	Route() string
	// Instance 返回后端驱动实例的名字。
	Instance() (string, error)
	// FanState 读取当前风扇运行模式与状态。
	FanState() (domain.FanState, error)
	// SetFans 设置风扇运行模式与转速占空比。
	SetFans(config domain.FanConfig) error
	// SetCoolBoost 开启或关闭酷冷强冷加速。
	SetCoolBoost(enabled bool) error
	// Lighting 读取当前键盘 RGB 灯效配置。
	Lighting() (domain.RGBConfig, error)
	// SetLighting 设置键盘 RGB 灯效配置。
	SetLighting(config domain.RGBConfig) error
	// ResetLighting 将键盘灯效恢复为出厂默认设置。
	ResetLighting() error
	// BacklightTimeout 读取键盘背光是否在 30 秒空闲时自动熄灭。
	BacklightTimeout() (bool, error)
	// SetBacklightTimeout 设置键盘背光自动熄灭。
	SetBacklightTimeout(enabled bool) error
	// GPUMode 查询当前的显卡工作模式。
	GPUMode() (domain.GPUMode, error)
	// SetGPUMode 切换显卡模式（混合/独显直连/自动）。
	SetGPUMode(mode domain.GPUMode) error
	// DiscreteGPU 查询当前是否工作在纯独显直连模式。
	DiscreteGPU() (bool, error)
	// LCDOverdrive 查询屏幕响应时间超速加速功能是否开启。
	LCDOverdrive() (bool, error)
	// SetLCDOverdrive 设置屏幕 LCD Overdrive 响应时间加速开关。
	SetLCDOverdrive(enabled bool) error
	// WinKeyLocked 查询 Windows 徽标键是否处于锁定禁用状态。
	WinKeyLocked() (bool, error)
	// SetWinKeyLocked 启用或禁用 Windows 徽标键锁定。
	SetWinKeyLocked(locked bool) error
	// BootSound 查询开机启动音效是否开启。
	BootSound() (bool, error)
	// SetBootSound 设置开机启动音效开关。
	SetBootSound(enabled bool) error
}

// AppearanceStore 保存界面的外观模式。config.Store 满足这个接口。
type AppearanceStore interface {
	// Theme 返回保存的外观模式。
	Theme() string
	// SetTheme 保存外观模式。
	SetTheme(mode string) error
}

// action 是排队等着执行的一次写入。
type action struct {
	run     func() error
	onError func(error)
}

// Controller 是界面与硬件之间唯一的关口。
//
// 它把机器的状态收成一份快照供界面直接读取，写入则排队交给后台协程，
// 因此界面线程永远不会被一次硬件写入卡住。
type Controller struct {
	hw    Hardware
	store AppearanceStore

	mu   sync.RWMutex
	snap model.Overview
	// err 是最近一次失败的原因；一次成功的读取会把它抹掉。
	err string
	// history 是最近的采样，界面上用来画曲线。
	history []model.Sample
	// smoother 为监控指标提供平滑阻尼缓冲。
	smoother *model.Smoother
	// smoothing 标记是否启用平滑缓动（后台运行时启用）。
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

// New 创建控制器，并做一次初始读取，让界面一上来就有东西可画。
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

// Snapshot 返回最近一次刷新的展示模型。它只读内存，不碰硬件，
// 因此界面线程调用它不会等待。
func (c *Controller) Snapshot() model.Overview {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snap
}

// Refresh 重新读取硬件状态并更新快照。它会等待原厂服务，
// 应当放在后台协程里调用。
//
// 读机器的时候不拿锁：界面线程读快照不能被一次慢读数拖住。
func (c *Controller) Refresh() {
	overview, monitor, failure := c.read()

	c.mu.Lock()
	defer c.mu.Unlock()
	if failure != "" {
		c.err = failure
	} else if c.err != "" && overview.Error == "" {
		// 上一次写入失败的原因接着摆着，直到下一次写入成功。
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

	// 读到数了，就把这次真实采样接到历史后面。
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

// read 读一次机器，把结果变成展示模型。
//
// 返回展示模型、这次读到的遥测（读失败时是 nil）与失败的原因。
// 它不碰控制器自己的状态，因此不必拿锁。
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
		// 档位读不出来不算整台机器坏了，接着往下读。
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

// Work 执行排队的写入动作，每执行一个就刷新一次快照。
// 后台协程按周期调用它；测试也可以直接调用它，不必等真实的时间。
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

// StepSmooth 步进一步平滑插值。如果有数值正在过渡，返回 true 并更新快照；
// 若已经全部稳定收敛，返回 false。
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

// SetSmoothing 开启或关闭读数平滑缓冲。
func (c *Controller) SetSmoothing(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.smoothing = enabled
}

// Start 启动后台协程：它按周期刷新快照，高频步进平滑过渡，执行排队的写入，
// 并在有变化时叫一声 onChange，让界面重画。onChange 可以是空。
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

// Stop 停下后台协程。
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

// SetPowerMode 排队切换性能档位。它不等待硬件，队列满了就返回 false，
// 那时界面该告诉用户机器正忙。失败时调用 onError。
func (c *Controller) SetPowerMode(mode domain.PowerMode, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetPowerMode(mode) }, onError)
}

// SetFanConfig 排队应用风扇运转模式与占空比配置。
//
// 针对自定义模式下连续拖动风扇占空比滑块的场景，采用单在途 + 最新待决合并机制：
// 如果底层正在执行上一次写入，新配置原子覆盖 pending 变量，
// 待底层写入返回后自动提取最新配置继续写入，避免在通道中堆积过期的中间状态，
// 从源头彻底消除高频写入造成的延迟卡顿与服务报错。
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

// SetFanConfigDebounced 在防抖延迟后排队应用风扇配置。
//
// 适合滑块连续拖动等高频输入场景：拖动期间每次变动重置等待，
// 停顿后才发送最终目标值，避免高频请求塞爆原厂驱动队列造成延迟卡壳。
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

// SetCoolBoost 排队切换酷冷强冷加速开关。
func (c *Controller) SetCoolBoost(enabled bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetCoolBoost(enabled) }, onError)
}

// SetLighting 排队应用键盘灯效配置。
//
// 针对滑块连续拖动与调色盘取色场景，采用单在途 + 最新待决合并机制：
// 如果底层正在执行上一次写入，新配置原子覆盖 pending 变量，
// 待底层写入返回后自动提取最新配置继续写入，避免在通道中堆积过期的中间状态，
// 从源头彻底消除高频写入造成的延迟卡顿与服务报错。
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

// SetLightingDebounced 在防抖延迟后排队应用键盘灯效配置。
//
// 适合滑块拖动或连续调色场景：变动期间重置计时，停顿后下发，避免请求堵塞硬件队列。
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

// ResetLighting 排队重置键盘灯效为默认出厂设置。
func (c *Controller) ResetLighting(onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.ResetLighting() }, onError)
}

// SetBacklightTimeout 排队设置键盘背光 30 秒超时自动休眠。
func (c *Controller) SetBacklightTimeout(enabled bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetBacklightTimeout(enabled) }, onError)
}

// SetGPUMode 排队切换显卡模式（混合/独显直连/自动）。
func (c *Controller) SetGPUMode(mode domain.GPUMode, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetGPUMode(mode) }, onError)
}

// SetLCDOverdrive 排队设置屏幕响应时间超速加速开关。
func (c *Controller) SetLCDOverdrive(enabled bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetLCDOverdrive(enabled) }, onError)
}

// SetWinKeyLocked 排队设置 Windows 徽标键锁定开关。
func (c *Controller) SetWinKeyLocked(locked bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetWinKeyLocked(locked) }, onError)
}

// SetBootSound 排队设置开机启动音效开关。
func (c *Controller) SetBootSound(enabled bool, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetBootSound(enabled) }, onError)
}

// enqueue 把一次写入放进队列。
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

// Samples 返回已经攒下的采样个数。
func (c *Controller) Samples() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.history)
}

// ThemeMode 返回当前保存的外观模式；没有保存过或存的不是认识的值时，
// 按自动处理。
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

// SetThemeMode 保存外观模式。空模式与不认识的值都不接受。
func (c *Controller) SetThemeMode(mode theme.Mode) error {
	if !mode.Valid() {
		return fmt.Errorf("controller: 不认识的外观模式 %q", mode)
	}
	if c.store == nil {
		return nil
	}
	return c.store.SetTheme(string(mode))
}
