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

	fanTimerMu sync.Mutex
	fanTimer   *time.Timer

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
			c.Refresh()
		default:
			return
		}
	}
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
	})
}

// SetPowerMode 排队切换性能档位。它不等待硬件，队列满了就返回 false，
// 那时界面该告诉用户机器正忙。失败时调用 onError。
func (c *Controller) SetPowerMode(mode domain.PowerMode, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetPowerMode(mode) }, onError)
}

// SetFanConfig 排队应用风扇运转模式与占空比配置。
func (c *Controller) SetFanConfig(config domain.FanConfig, onError func(error)) bool {
	return c.enqueue(func() error { return c.hw.SetFans(config) }, onError)
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
