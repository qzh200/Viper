// Package app is the use case layer of Viper.
//
// It joins the hardware backend with the stored settings. The settings that
// do not belong to the gaming hardware, such as the blue light shield, are
// read from the service of the original software.
//
// 这一层现在接的还是固件那一侧（internal/hardware），所以灯光的颜色与灯效
// 还是从设置文件里读的。硬件层的新路线（internal/machine，原厂服务当主路）
// 已经做好，接线是下一步的事：接上之后，灯光直接读机器真正的样子，原厂服务
// 读得到分区颜色与灯效，固件那一侧读不到。
package app

import (
	"fmt"
	"sync"

	"viper/internal/config"
	"viper/internal/hardware"
	"viper/internal/machine"
	"viper/internal/nvidia"
	"viper/internal/quickaccess"
)

// abilities is the set of abilities of the machine that the program uses.
//
// The program does not report the table of the fans, so it depends on the
// small interfaces instead of the whole of the machine.
type abilities interface {
	hardware.ReadingsReader
	hardware.PowerModeReader
	hardware.GraphicsModeReader
	hardware.LightingReader
	hardware.LightingConfig
	hardware.LightingWriter
	hardware.LightingReset
	hardware.FansReader
	hardware.CoolBoostReader
	hardware.SwitchesReader
}

// The routes the program can reach the machine by.
const (
	// routeService goes through the service of the original software. It is
	// the main road and it needs no rights of an administrator.
	routeService = "原厂服务 AcerAgentService"

	// routeFirmware goes through the ACPI provider. It is the road of the
	// few settings the service does not know, and it needs the rights of
	// an administrator.
	routeFirmware = "固件接口 root\\wmi\\AcerGamingFunction"
)

// App is the running program.
type App struct {
	// mu 看守着程序按需打开的连接。游戏硬件和 Quick Access 服务都是
	// 第一次用到相关设置时才打开，所以不需要管理员权限的设置不会去
	// 碰硬件。这把锁保证同时到来的两次调用不会各开一条连接，然后把
	// 其中一条漏掉。
	mu sync.Mutex
	// backend 是到游戏硬件的连接。
	backend abilities
	// openBackend 打开到机器的连接。它是可变的，所以用例可以放一个替身
	// 在真机器的位置上。
	openBackend func() (hardware.Gaming, string, error)
	// route 记着程序走的是哪条路：原厂服务还是固件那一侧。
	route string
	// closed 记录程序是否已经关闭。关闭之后的程序不再打开硬件。
	closed bool
	// store 保存两次运行之间的设置。
	store *config.Store
	// quickClient 是与 Quick Access 服务的会话。它由测试注入，程序
	// 自己每次调用都开一个新的会话。
	quickClient QuickAccess
	// quickOptions change the behaviour of the session with the Quick
	// Access service.
	quickOptions quickaccess.Options
}

// QuickAccess is the set of abilities of the Quick Access service that
// the program uses.
//
// The interface keeps the program testable: a test can hand the program
// a stand for the service, so that the behaviour can be checked without
// the service of the original software.
type QuickAccess interface {
	// BlueLightShield reads the level of the blue light shield.
	BlueLightShield() (quickaccess.BlueLight, error)
	// SetBlueLightShield selects a level of the blue light shield.
	SetBlueLightShield(level quickaccess.BlueLight) error
	// USBChargeLimit reads the charging limit in percent.
	USBChargeLimit() (int, error)
	// SetUSBChargeLimit sets the charging limit in percent.
	SetUSBChargeLimit(percent int) error
	// USBChargeEnabled reports whether the charging over USB is allowed.
	USBChargeEnabled() (bool, error)
	// SetUSBChargeEnabled allows or forbids the charging over USB.
	SetUSBChargeEnabled(enabled bool) error
	// DisplayMode 读取机器的显卡切换方式。
	DisplayMode() (quickaccess.DisplayMode, error)
	// SetDisplayMode 选择机器的显卡切换方式。
	SetDisplayMode(mode quickaccess.DisplayMode) error
	// DarkMode 报告系统的深色模式是否打开。
	DarkMode() (bool, error)
	// SetDarkMode 打开或关闭系统的深色模式。
	SetDarkMode(enabled bool) error
	// TransparentEffect 报告系统的窗口是不是透明的。
	TransparentEffect() (bool, error)
	// SetTransparentEffect 打开或关闭窗口的透明效果。
	SetTransparentEffect(enabled bool) error
	// AutoHideTaskbar 报告任务栏是不是自己藏起来。
	AutoHideTaskbar() (bool, error)
	// SetAutoHideTaskbar 打开或关闭任务栏的自动隐藏。
	SetAutoHideTaskbar(enabled bool) error
	// ScreenSaver 报告屏幕保护程序是不是打开的。
	ScreenSaver() (bool, error)
	// SetScreenSaver 打开或关闭屏幕保护程序。
	SetScreenSaver(enabled bool) error
	// ScreenSaverTimeout 读取屏幕保护程序启动前的空闲时间，单位是秒。
	ScreenSaverTimeout() (int, error)
	// SetScreenSaverTimeout 设置屏幕保护程序启动前的空闲时间，单位是秒。
	SetScreenSaverTimeout(seconds int) error
	// Close ends the session.
	Close() error
}

// New opens the stored settings.
//
// The connection to the gaming hardware is opened on first use, the
// settings of the original software do not need the rights that the
// hardware needs.
func New() (*App, error) {
	store, err := config.Open()
	if err != nil {
		return nil, err
	}
	return NewWithStore(store), nil
}

// NewWithStore opens the program with the given settings.
//
// It is used by the tests, so that they do not touch the settings of the
// user.
func NewWithStore(store *config.Store) *App {
	return &App{store: store}
}

// NewWithBackend opens the program with the given hardware.
//
// It is used by the tests, so that they do not need the rights of an
// administrator to talk to the firmware.
func NewWithBackend(store *config.Store, backend abilities) *App {
	return &App{store: store, backend: backend}
}

// NewWithQuickAccess opens the program with the given session of the
// Quick Access service.
//
// It is used by the tests, so that they do not need the service of the
// original software.
func NewWithQuickAccess(store *config.Store, client QuickAccess) *App {
	return &App{store: store, quickClient: client}
}

// SetQuickAccessOptions changes the way the program opens a session with
// the Quick Access service.
//
// It has to be called before the first session is opened, an address
// without the service of the original software makes every call of the
// service fail.
//
// 这个入口让 quickOptions 不再是一个永远为零的字段。
func (a *App) SetQuickAccessOptions(options quickaccess.Options) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.quickOptions = options
}

// Close releases the hardware.
func (a *App) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.backend != nil {
		a.backend.Close()
		a.backend = nil
	}
	a.closed = true
}

// hardware opens the connection to the gaming hardware.
//
// The connection is only made when a hardware setting is used, because
// opening it takes a moment and the settings of the original software do
// not need it.
func (a *App) hardware() (abilities, error) {
	return a.connect(func() (abilities, error) {
		open := a.openBackend
		if open == nil {
			open = openMachine
		}

		backend, route, err := open()
		if err != nil {
			return nil, err
		}
		a.route = route
		return backend, nil
	})
}

// openMachine opens the connection to the machine.
//
// 主路是原厂服务：它以系统的身份运行，程序不用管理员权限就能用它，而且它
// 读得到分区颜色与灯效，固件那一侧读不到。原厂服务不在的时候退回固件那
// 一侧，那一侧要管理员权限。
//
// 第二个结果是程序走的是哪条路，命令行会把它报出来，免得用的人不知道
// 自己用的是哪一条。
func openMachine() (hardware.Gaming, string, error) {
	backend, err := machine.New(machine.Options{
		OpenFirmware: hardware.OpenFirmware,
	})
	if err == nil {
		return backend, routeService, nil
	}

	firmware, firmwareErr := hardware.NewBackend()
	if firmwareErr != nil {
		return nil, "", fmt.Errorf("原厂服务不可达（%v），固件接口也连不上: %w", err, firmwareErr)
	}
	return firmware, routeFirmware, nil
}

// SetBackendOpener changes the way the program opens the connection to the
// machine.
//
// It has to be called before the first connection is opened. The second
// result is the name of the route the program is on, it is reported to the
// user so that they know which road they are on.
//
// 这个入口让用例可以把一台机器的替身放在真机器的位置上，不用去碰真的
// 机器。
func (a *App) SetBackendOpener(open func() (hardware.Gaming, string, error)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.openBackend = open
}

// Route returns the way the program reaches the machine.
//
// The connection is opened on first use, so before that the answer is that
// the route is not known yet.
func (a *App) Route() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.route == "" {
		return "还没连接"
	}
	return a.route
}

// connect opens the connection to the gaming hardware once.
//
// 连接由锁守着，同时到来的两次调用不会开出两条连接然后漏掉其中一条。
func (a *App) connect(open func() (abilities, error)) (abilities, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil, fmt.Errorf("app: 程序已经关闭")
	}
	if a.backend != nil {
		return a.backend, nil
	}
	backend, err := open()
	if err != nil {
		return nil, fmt.Errorf("app: 打开游戏硬件失败: %w", err)
	}
	a.backend = backend
	return backend, nil
}

// Instance returns the object path of the gaming functions.
func (a *App) Instance() (string, error) {
	backend, err := a.hardware()
	if err != nil {
		return "", err
	}
	return backend.Instance(), nil
}

// SettingsPath returns the location of the settings file.
func (a *App) SettingsPath() string { return a.store.Path() }

// Sensors reads the live temperatures and fan speeds.
func (a *App) Sensors() (hardware.Sensors, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.Sensors{}, err
	}
	return backend.Sensors()
}

// PowerStatus reads the power supply of the machine.
//
// The information comes from the kernel of Windows, so it needs neither the
// gaming provider nor the services of the original software.
func (a *App) PowerStatus() (hardware.PowerStatus, error) {
	return hardware.SystemPowerStatus()
}

// GPUTelemetry reads the state of the graphics card.
//
// The numbers come from the driver of the card. A machine without a card
// that offers them answers with an empty list.
func (a *App) GPUTelemetry() ([]nvidia.Telemetry, error) {
	cards, err := nvidia.Read()
	if err != nil {
		// A machine without the driver of the vendor simply has no
		// telemetry, that is not an error.
		if nvidia.Available() {
			return nil, err
		}
		return nil, nil
	}
	return cards, nil
}

// Lighting returns the keyboard lighting.
//
// 原厂服务读得回分区颜色与灯效，所以那一路给的是机器真正的样子。固件那一
// 侧只报亮度，颜色与灯效就只剩 Viper 自己记着的那一份了。
//
// 读不到亮度的时候这次调用直接失败，调用方必须能分清“固件读不到”和
// “亮度就是零”，所以这里不再拿设置文件里的旧亮度顶替。
func (a *App) Lighting() (hardware.RGBConfig, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.RGBConfig{}, err
	}

	// 原厂服务这一路能读回颜色与灯效。
	result, err := backend.Lighting()
	if err == nil && result.ColourReadable && result.EffectReadable {
		return result, nil
	}

	// 固件只报告亮度，灯效和颜色存在 Viper 自己的设置里。
	stored := a.StoredLighting()
	stored.ColourReadable = false
	stored.EffectReadable = false

	brightness, err := backend.LightingBrightness()
	if err != nil {
		return hardware.RGBConfig{}, fmt.Errorf("app: 读取键盘亮度失败: %w", err)
	}
	stored.Brightness = brightness
	return stored, nil
}

// StoredLighting returns the keyboard lighting that Viper keeps.
//
// The firmware reports the brightness only, the effect and the colours of
// the machine live in the settings of Viper. This method gives them back
// without asking the firmware, so that a program can still show the
// lighting of a machine whose firmware cannot be reached.
//
// 固件只报告亮度，灯效和颜色存在 Viper 自己的设置里。这个方法不去问
// 固件，所以固件连不上的时候也能用。
func (a *App) StoredLighting() hardware.RGBConfig {
	stored := a.store.Settings().Lighting

	result := hardware.RGBConfig{
		Effect:     hardware.LightingEffect(stored.Effect),
		Speed:      stored.Speed,
		Direction:  stored.Direction,
		Brightness: stored.Brightness,
		PerZone:    stored.PerZone,
	}
	for i, colour := range stored.Zones {
		result.Zones[i] = hardware.RGBColor{R: colour.R, G: colour.G, B: colour.B}
	}
	return result
}

// SetLighting applies a keyboard lighting configuration.
//
// The values are limited to the ranges that the firmware accepts before
// they are written, so that the firmware and the stored settings always
// carry the same numbers.
//
// 写进固件之前先钳制速度、方向和亮度，写进固件的值和记下来的值因此
// 一定是同一个值。
func (a *App) SetLighting(value hardware.RGBConfig) error {
	value = clampLighting(value)

	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetLighting(value); err != nil {
		return fmt.Errorf("app: 设置键盘灯效失败: %w", err)
	}

	stored := toStoredLighting(value)
	if err := a.store.Update(func(s *config.Settings) {
		s.Lighting = stored
	}); err != nil {
		return fmt.Errorf("app: 保存键盘灯效设置失败: %w", err)
	}
	return nil
}

// clampLighting limits a lighting configuration to the values that the
// firmware accepts.
//
// The brightness runs from zero to one hundred, the speed from one to nine
// and the direction is a switch. The limits come from the profile of the
// original software.
//
// 亮度是零到一百，速度是一到九，方向是一个开关。
func clampLighting(value hardware.RGBConfig) hardware.RGBConfig {
	value.Speed = clamp(value.Speed, 1, 9)
	value.Direction = clamp(value.Direction, 0, 1)
	value.Brightness = clamp(value.Brightness, 0, 100)
	return value
}

// toStoredLighting converts a lighting configuration into its stored form.
//
// The values are limited to the range that the firmware accepts, so that
// the stored settings never claim more than the keyboard can do. The
// limits come from the profile of the original software: the brightness
// runs from zero to one hundred, the speed from one to nine and the
// direction is a switch.
func toStoredLighting(value hardware.RGBConfig) config.Lighting {
	value = clampLighting(value)
	stored := config.Lighting{
		Effect:     uint64(value.Effect),
		Speed:      value.Speed,
		Direction:  value.Direction,
		Brightness: value.Brightness,
		PerZone:    value.PerZone,
	}
	for i, colour := range value.Zones {
		stored.Zones[i] = config.LightingColor{R: colour.R, G: colour.G, B: colour.B}
	}
	return stored
}

// clamp limits a number to the given range.
func clamp(value uint64, low, high uint64) uint64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// ResetLighting restores the default keyboard lighting.
func (a *App) ResetLighting() error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.ResetLighting(); err != nil {
		return fmt.Errorf("app: 恢复默认键盘灯效失败: %w", err)
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.Lighting = config.Lighting{
			Effect:     uint64(hardware.EffectStatic),
			Speed:      3,
			Direction:  0,
			Brightness: 100,
			PerZone:    true,
		}
		for i := range s.Lighting.Zones {
			s.Lighting.Zones[i] = config.LightingColor{R: 255, G: 255, B: 255}
		}
	}); err != nil {
		return fmt.Errorf("app: 保存键盘灯效设置失败: %w", err)
	}
	return nil
}

// FanStatus is the fan configuration of the machine.
//
// The provider reports the behaviour of the group of the processor only,
// the group of the graphics fan cannot be read back. The mode of the
// graphics fan is therefore the mode that Viper wrote last, the flag
// GPUModeReadable tells the caller that this number did not come from the
// firmware.
//
// 固件只报告处理器这一组的行为，显卡这一组读不回来。显卡风扇的模式取自
// Viper 最后一次写入的值，GPUModeReadable 说明这个值不是从固件读来的。
type FanStatus struct {
	hardware.FanState
	// GPUModeReadable 报告显卡风扇的模式是不是从固件读来的。固件不
	// 报告这一组，所以每一台机器上它都是假。
	GPUModeReadable bool
}

// Fans returns the fan configuration.
func (a *App) Fans() (FanStatus, error) {
	backend, err := a.hardware()
	if err != nil {
		return FanStatus{}, err
	}
	state, err := backend.Fans()
	if err != nil {
		return FanStatus{}, fmt.Errorf("app: 读取风扇设置失败: %w", err)
	}

	status := FanStatus{FanState: state}
	// 固件不报告显卡这一组，模式取自 Viper 最后一次写入的值，并且标记成
	// 不是从固件读来的。原厂服务读得回这一组，就用它自己的读数。
	if !state.GPUModeReadable {
		status.GPUMode = fanMode(a.store.Settings().Fans.GPUMode)
		status.GPUModeReadable = false
	} else {
		status.GPUModeReadable = true
	}
	return status, nil
}

// SetFans applies a fan configuration and remembers it.
func (a *App) SetFans(fans hardware.FanConfig) error {
	if !knownFanMode(fans.CPUMode) {
		return fmt.Errorf("app: 处理器风扇的行为 %q 不认识", fans.CPUMode)
	}
	if !knownFanMode(fans.GPUMode) {
		return fmt.Errorf("app: 显卡风扇的行为 %q 不认识", fans.GPUMode)
	}

	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetFans(fans); err != nil {
		return fmt.Errorf("app: 设置风扇失败: %w", err)
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.Fans = config.Fans{
			CPUMode:    storedFanMode(fans.CPUMode),
			GPUMode:    storedFanMode(fans.GPUMode),
			CPUPercent: fans.CPUPercent,
			GPUPercent: fans.GPUPercent,
			CoolBoost:  fans.CoolBoost,
		}
	}); err != nil {
		return fmt.Errorf("app: 保存风扇设置失败: %w", err)
	}
	return nil
}

// knownFanMode reports whether a behaviour of a fan group is part of the
// enumeration of the hardware layer.
//
// 行为必须是硬件层枚举里的一个。
func knownFanMode(mode hardware.FanMode) bool {
	switch mode {
	case hardware.FanAuto, hardware.FanMax, hardware.FanCustom:
		return true
	default:
		return false
	}
}

// fanMode maps a stored behaviour onto the enumeration of the hardware
// layer.
//
// The settings hold the semantic enumeration of the hardware layer. A
// value that is not part of it comes from a file that was written by hand
// or by another tool, such a value is treated as the automatic mode, the
// curve of the embedded controller.
//
// 设置文件里存的是语义枚举。文件里的值不在枚举里，说明它被人手改过或者
// 是别的工具写的，这种值按自动模式处理。
func fanMode(value uint64) hardware.FanMode {
	switch value {
	case config.FanMax:
		return hardware.FanMax
	case config.FanCustom:
		return hardware.FanCustom
	default:
		return hardware.FanAuto
	}
}

// storedFanMode maps a behaviour of the hardware layer onto the value
// that is stored.
//
// The settings carry an enumeration of their own, so that the file keeps
// its meaning even when the hardware layer gets a new behaviour.
//
// 设置文件带着自己的枚举，这样硬件层以后多一种行为时，文件里的意思不会
// 跟着变。
func storedFanMode(mode hardware.FanMode) uint64 {
	switch mode {
	case hardware.FanMax:
		return config.FanMax
	case hardware.FanCustom:
		return config.FanCustom
	default:
		return config.FanAuto
	}
}

// powerMode maps a stored profile onto the enumeration of the hardware
// layer.
//
// The settings hold the semantic enumeration of the hardware layer. A
// value that is not part of it comes from a file that was written by hand
// or by another tool, such a value is treated as the balanced profile.
//
// 设置文件里存的是语义枚举。文件里的值不在枚举里，说明它被人手改过或者
// 是别的工具写的，这种值按均衡模式处理。
func powerMode(value uint64) hardware.PowerMode {
	switch value {
	case config.PowerQuiet:
		return hardware.PowerQuiet
	case config.PowerPerformance:
		return hardware.PowerPerformance
	case config.PowerTurbo:
		return hardware.PowerTurbo
	case config.PowerUnknown:
		return hardware.PowerUnknown
	case config.PowerEco:
		return hardware.PowerEco
	default:
		return hardware.PowerBalanced
	}
}

// storedPowerMode maps a profile of the hardware layer onto the value that
// is stored.
//
// The settings carry an enumeration of their own, so that the file keeps
// its meaning even when the hardware layer gets a new profile.
//
// 设置文件带着自己的枚举，这样硬件层以后多一个档位时，文件里的意思不会
// 跟着变。
func storedPowerMode(mode hardware.PowerMode) uint64 {
	switch mode {
	case hardware.PowerQuiet:
		return config.PowerQuiet
	case hardware.PowerPerformance:
		return config.PowerPerformance
	case hardware.PowerTurbo:
		return config.PowerTurbo
	case hardware.PowerUnknown:
		return config.PowerUnknown
	case hardware.PowerEco:
		return config.PowerEco
	default:
		return config.PowerBalanced
	}
}

// CoolBoost reports whether the boost fan table is selected.
func (a *App) CoolBoost() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.CoolBoostEnabled()
}

// SetCoolBoost switches the boost fan table.
func (a *App) SetCoolBoost(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.CoolBoost(enabled); err != nil {
		return fmt.Errorf("app: 设置风扇加速失败: %w", err)
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.Fans.CoolBoost = enabled
	}); err != nil {
		return fmt.Errorf("app: 保存风扇加速设置失败: %w", err)
	}
	return nil
}

// PowerMode returns the performance profile.
func (a *App) PowerMode() (hardware.PowerMode, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.PowerBalanced, err
	}
	return backend.PowerMode()
}

// SetPowerMode applies a performance profile and remembers it.
//
// When a scene is bound to the profile, the scene is applied with it, so
// that every profile brings its own lighting and its own fans.
func (a *App) SetPowerMode(mode hardware.PowerMode) error {
	if mode != hardware.PowerQuiet && mode != hardware.PowerBalanced &&
		mode != hardware.PowerPerformance && mode != hardware.PowerTurbo &&
		mode != hardware.PowerEco {
		return fmt.Errorf("app: 性能档位 %q 不认识", mode)
	}

	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetPowerMode(mode); err != nil {
		return fmt.Errorf("app: 设置性能档位失败: %w", err)
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.PowerMode = storedPowerMode(mode)
	}); err != nil {
		return fmt.Errorf("app: 保存性能档位失败: %w", err)
	}
	// 记下来的值必须还能读回同一个档位，这一行守着两套编号不会走偏。
	if got := powerMode(a.store.Settings().PowerMode); got != mode {
		return fmt.Errorf("app: 性能档位 %s 记下来之后变成了 %s", mode, got)
	}

	settings := a.store.Settings()
	scene, ok := settings.Bindings[mode.String()]
	if !ok {
		return nil
	}
	scenario, ok := settings.Scenarios[scene]
	if !ok {
		return nil
	}
	return a.applyScenario(scenario)
}

// Scenarios returns the saved scenes.
//
// The scenes are a copy, the caller cannot change the stored scenes by
// accident.
//
// 返回的是副本，调用方改不动里面的场景。
func (a *App) Scenarios() map[string]config.Scenario {
	return a.store.Settings().Scenarios
}

// Bindings returns the scenes that are bound to a performance profile.
//
// The bindings are a copy, the caller cannot change the stored bindings
// by accident.
//
// 返回的是副本，调用方改不动里面的绑定。
func (a *App) Bindings() map[string]string {
	return a.store.Settings().Bindings
}

// SaveScenario stores the current lighting and fans under a name.
//
// The fan configuration and the brightness are read from the machine, so
// the scene holds what the hardware really does. The effect and the
// colours cannot be read back, they are taken from the settings of Viper.
func (a *App) SaveScenario(name string) error {
	if name == "" {
		return fmt.Errorf("app: 场景需要一个名字")
	}

	settings := a.store.Settings()
	scenario := config.Scenario{
		Lighting: settings.Lighting,
		Fans:     settings.Fans,
	}

	if state, err := a.Fans(); err == nil {
		scenario.Fans = config.Fans{
			CPUMode:    storedFanMode(state.CPUMode),
			GPUMode:    storedFanMode(state.GPUMode),
			CPUPercent: state.CPUPercent,
			GPUPercent: state.GPUPercent,
			CoolBoost:  state.CoolBoost,
		}
	}

	if backend, err := a.hardware(); err == nil {
		// The firmware reports a level of zero while the backlight has
		// switched itself off, so a zero is not taken for the level the
		// user has chosen.
		if brightness, err := backend.LightingBrightness(); err == nil && brightness > 0 {
			scenario.Lighting.Brightness = brightness
		}
	}

	if err := a.store.Update(func(s *config.Settings) {
		s.Scenarios[name] = scenario
	}); err != nil {
		return fmt.Errorf("app: 保存场景 %q 失败: %w", name, err)
	}
	return nil
}

// DeleteScenario removes a scene and the bindings that point to it.
func (a *App) DeleteScenario(name string) error {
	if err := a.store.Update(func(s *config.Settings) {
		delete(s.Scenarios, name)
		for profile, scene := range s.Bindings {
			if scene == name {
				delete(s.Bindings, profile)
			}
		}
	}); err != nil {
		return fmt.Errorf("app: 删除场景 %q 失败: %w", name, err)
	}
	return nil
}

// ApplyScenario applies a scene to the machine.
func (a *App) ApplyScenario(name string) error {
	scenario, ok := a.store.Settings().Scenarios[name]
	if !ok {
		return fmt.Errorf("app: 没有叫 %q 的场景", name)
	}
	return a.applyScenario(scenario)
}

// applyScenario writes a scene to the machine and remembers it.
func (a *App) applyScenario(scenario config.Scenario) error {
	lighting := hardware.RGBConfig{
		Effect:     hardware.LightingEffect(scenario.Lighting.Effect),
		Speed:      scenario.Lighting.Speed,
		Direction:  scenario.Lighting.Direction,
		Brightness: scenario.Lighting.Brightness,
		PerZone:    scenario.Lighting.PerZone,
	}
	for i, colour := range scenario.Lighting.Zones {
		lighting.Zones[i] = hardware.RGBColor{R: colour.R, G: colour.G, B: colour.B}
	}
	if err := a.SetLighting(lighting); err != nil {
		return err
	}

	return a.SetFans(hardware.FanConfig{
		CPUMode:    fanMode(scenario.Fans.CPUMode),
		GPUMode:    fanMode(scenario.Fans.GPUMode),
		CPUPercent: scenario.Fans.CPUPercent,
		GPUPercent: scenario.Fans.GPUPercent,
		CoolBoost:  scenario.Fans.CoolBoost,
	})
}

// BindScenario ties a performance profile to a scene.
func (a *App) BindScenario(profile, scene string) error {
	if _, ok := a.store.Settings().Scenarios[scene]; !ok {
		return fmt.Errorf("app: 没有叫 %q 的场景", scene)
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.Bindings[profile] = scene
	}); err != nil {
		return fmt.Errorf("app: 保存档位 %q 的场景绑定失败: %w", profile, err)
	}
	return nil
}

// UnbindScenario removes the scene of a performance profile.
func (a *App) UnbindScenario(profile string) error {
	if err := a.store.Update(func(s *config.Settings) {
		delete(s.Bindings, profile)
	}); err != nil {
		return fmt.Errorf("app: 删除档位 %q 的场景绑定失败: %w", profile, err)
	}
	return nil
}

// AvailablePowerModes lists the profiles this machine supports.
func (a *App) AvailablePowerModes() []hardware.PowerMode {
	return hardware.AvailablePowerModes(hardware.IsPredator())
}

// Identity returns the machine information from the firmware.
func (a *App) Identity() (hardware.Identity, error) {
	return hardware.SystemInfo()
}

// BacklightTimeout reports whether the backlight switches off on its own.
func (a *App) BacklightTimeout() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	backlight, err := backend.Backlight()
	if err != nil {
		return false, err
	}
	return backlight.Timeout, nil
}

// SetBacklightTimeout switches the automatic switch off on its own.
func (a *App) SetBacklightTimeout(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	brightness, err := backend.LightingBrightness()
	if err != nil {
		return fmt.Errorf("app: 读取键盘亮度失败: %w", err)
	}
	if err := backend.SetBacklight(int(brightness), enabled); err != nil {
		return fmt.Errorf("app: 设置背光自动熄灭失败: %w", err)
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.BacklightTimeout = enabled
	}); err != nil {
		return fmt.Errorf("app: 保存背光自动熄灭设置失败: %w", err)
	}
	return nil
}

// LCDOverdrive reports whether the display overdrive is active.
func (a *App) LCDOverdrive() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.LCDOverdrive()
}

// SetLCDOverdrive enables or disables the display overdrive.
func (a *App) SetLCDOverdrive(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetLCDOverdrive(enabled); err != nil {
		return fmt.Errorf("app: 设置屏幕超频失败: %w", err)
	}
	return nil
}

// WinKeyLocked reports whether the Windows key is locked.
func (a *App) WinKeyLocked() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.WinKeyLocked()
}

// SetWinKeyLocked locks or unlocks the Windows key.
func (a *App) SetWinKeyLocked(locked bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetWinKeyLocked(locked); err != nil {
		return fmt.Errorf("app: 设置 Windows 键锁定失败: %w", err)
	}
	return nil
}

// DiscreteGPU reports the state of the discrete GPU switch.
func (a *App) DiscreteGPU() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.DiscreteGPU()
}

// OverclockCapability reports which overclocking features the machine has.
func (a *App) OverclockCapability() (gpu bool, cpu bool, err error) {
	backend, err := a.hardware()
	if err != nil {
		return false, false, err
	}
	return backend.OverclockCapability()
}

// StickyKeys reports whether the sticky keys are switched on.
//
// The sticky keys belong to the accessibility settings of the current
// user, they are not part of the gaming hardware.
func (a *App) StickyKeys() (bool, error) { return hardware.StickyKeys() }

// SetStickyKeys switches the sticky keys on or off.
func (a *App) SetStickyKeys(enabled bool) error { return hardware.SetStickyKeys(enabled) }

// GPUMode reads the graphics mode of the machine.
func (a *App) GPUMode() (hardware.GPUMode, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.GPUModeHybrid, err
	}
	return backend.GPUMode()
}

// SetGPUMode selects the graphics mode of the machine.
func (a *App) SetGPUMode(mode hardware.GPUMode) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetGPUMode(mode); err != nil {
		return fmt.Errorf("app: 设置显卡模式失败: %w", err)
	}
	return nil
}

// BootSound reports whether the sound at the start of the machine is on.
func (a *App) BootSound() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.BootSound()
}

// SetBootSound switches the sound at the start of the machine on or off.
func (a *App) SetBootSound(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetBootSound(enabled); err != nil {
		return fmt.Errorf("app: 设置开机声音失败: %w", err)
	}
	return nil
}

// BlueLightShield reads the level of the blue light shield.
//
// The setting belongs to the Quick Access service of the original stack,
// it is not part of the gaming hardware provider.
func (a *App) BlueLightShield() (quickaccess.BlueLight, error) {
	client, release, err := a.quick()
	if err != nil {
		return quickaccess.BlueLightDisabled, err
	}
	defer release()
	level, err := client.BlueLightShield()
	if err != nil {
		return quickaccess.BlueLightDisabled, fmt.Errorf("app: 读取蓝光屏蔽失败: %w", err)
	}
	return level, nil
}

// SetBlueLightShield selects a level of the blue light shield.
func (a *App) SetBlueLightShield(level quickaccess.BlueLight) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetBlueLightShield(level); err != nil {
		return fmt.Errorf("app: 设置蓝光屏蔽失败: %w", err)
	}
	return nil
}

// USBChargeLimit reads the charging limit in percent.
func (a *App) USBChargeLimit() (int, error) {
	client, release, err := a.quick()
	if err != nil {
		return 0, err
	}
	defer release()
	limit, err := client.USBChargeLimit()
	if err != nil {
		return 0, fmt.Errorf("app: 读取充电上限失败: %w", err)
	}
	return limit, nil
}

// SetUSBChargeLimit sets the charging limit in percent.
func (a *App) SetUSBChargeLimit(percent int) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetUSBChargeLimit(percent); err != nil {
		return fmt.Errorf("app: 设置充电上限失败: %w", err)
	}
	return nil
}

// USBChargeEnabled reports whether the charging over USB is allowed.
func (a *App) USBChargeEnabled() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.USBChargeEnabled()
	if err != nil {
		return false, fmt.Errorf("app: 读取 USB 充电开关失败: %w", err)
	}
	return enabled, nil
}

// SetUSBChargeEnabled allows or forbids the charging over USB.
func (a *App) SetUSBChargeEnabled(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetUSBChargeEnabled(enabled); err != nil {
		return fmt.Errorf("app: 设置 USB 充电开关失败: %w", err)
	}
	return nil
}

// DisplayMode reads the graphics switch of the machine.
//
// The setting belongs to the Quick Access service of the original stack,
// it is not part of the gaming hardware provider.
func (a *App) DisplayMode() (quickaccess.DisplayMode, error) {
	client, release, err := a.quick()
	if err != nil {
		return quickaccess.DisplayModeHybrid, err
	}
	defer release()
	mode, err := client.DisplayMode()
	if err != nil {
		return quickaccess.DisplayModeHybrid, fmt.Errorf("app: 读取显卡切换方式失败: %w", err)
	}
	return mode, nil
}

// SetDisplayMode selects the graphics switch of the machine.
func (a *App) SetDisplayMode(mode quickaccess.DisplayMode) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetDisplayMode(mode); err != nil {
		return fmt.Errorf("app: 设置显卡切换方式失败: %w", err)
	}
	return nil
}

// DarkMode reports whether the dark mode of the system is switched on.
//
// The setting belongs to the Quick Access service of the original stack,
// it is not part of the gaming hardware provider.
func (a *App) DarkMode() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.DarkMode()
	if err != nil {
		return false, fmt.Errorf("app: 读取深色模式失败: %w", err)
	}
	return enabled, nil
}

// SetDarkMode switches the dark mode of the system on or off.
func (a *App) SetDarkMode(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetDarkMode(enabled); err != nil {
		return fmt.Errorf("app: 设置深色模式失败: %w", err)
	}
	return nil
}

// TransparentEffect reports whether the windows of the system are
// transparent.
//
// The ability belongs to the Quick Access service of the original stack.
// A service that does not know the command answers with an error, the
// program never answers that the effect is switched off.
func (a *App) TransparentEffect() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.TransparentEffect()
	if err != nil {
		return false, fmt.Errorf("app: 读取透明效果失败: %w", err)
	}
	return enabled, nil
}

// SetTransparentEffect switches the transparency of the windows.
func (a *App) SetTransparentEffect(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetTransparentEffect(enabled); err != nil {
		return fmt.Errorf("app: 设置透明效果失败: %w", err)
	}
	return nil
}

// AutoHideTaskbar reports whether the taskbar hides itself.
//
// The ability belongs to the Quick Access service of the original stack.
// A service that does not know the command answers with an error, the
// program never answers that the taskbar does not hide itself.
func (a *App) AutoHideTaskbar() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	hidden, err := client.AutoHideTaskbar()
	if err != nil {
		return false, fmt.Errorf("app: 读取任务栏自动隐藏失败: %w", err)
	}
	return hidden, nil
}

// SetAutoHideTaskbar switches the automatic hiding of the taskbar.
func (a *App) SetAutoHideTaskbar(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetAutoHideTaskbar(enabled); err != nil {
		return fmt.Errorf("app: 设置任务栏自动隐藏失败: %w", err)
	}
	return nil
}

// ScreenSaver reports whether the screen saver is switched on.
//
// The ability belongs to the Quick Access service of the original stack.
// A service that does not know the command answers with an error, the
// program never answers that the screen saver is switched off.
func (a *App) ScreenSaver() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.ScreenSaver()
	if err != nil {
		return false, fmt.Errorf("app: 读取屏幕保护程序失败: %w", err)
	}
	return enabled, nil
}

// SetScreenSaver switches the screen saver on or off.
func (a *App) SetScreenSaver(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetScreenSaver(enabled); err != nil {
		return fmt.Errorf("app: 设置屏幕保护程序失败: %w", err)
	}
	return nil
}

// ScreenSaverTimeout reads the idle time in seconds before the screen
// saver starts.
//
// The ability belongs to the Quick Access service of the original stack.
// A service that does not know the command answers with an error.
func (a *App) ScreenSaverTimeout() (int, error) {
	client, release, err := a.quick()
	if err != nil {
		return 0, err
	}
	defer release()
	timeout, err := client.ScreenSaverTimeout()
	if err != nil {
		return 0, fmt.Errorf("app: 读取屏幕保护程序超时失败: %w", err)
	}
	return timeout, nil
}

// SetScreenSaverTimeout sets the idle time in seconds before the screen
// saver starts.
func (a *App) SetScreenSaverTimeout(seconds int) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetScreenSaverTimeout(seconds); err != nil {
		return fmt.Errorf("app: 设置屏幕保护程序超时失败: %w", err)
	}
	return nil
}

// quick opens a session with the Quick Access service of the original
// stack.
//
// The service owns the settings that do not belong to the gaming
// hardware, such as the blue light shield and the charging over USB. It
// is not needed for the other settings, so the session is only opened
// when one of them is used.
//
// 注入给程序的会话属于注入的人，程序不会关掉它。程序自己开的会话每次
// 调用都用完就关，因为关掉的会话不能再用第二次。
//
// 返回的函数负责结束会话，用完之后必须调用。
func (a *App) quick() (QuickAccess, func(), error) {
	a.mu.Lock()
	client := a.quickClient
	options := a.quickOptions
	a.mu.Unlock()

	if client != nil {
		return client, func() {}, nil
	}

	owned, err := quickaccess.Connect(options)
	if err != nil {
		return nil, nil, fmt.Errorf("app: 连不上原厂软件的 Quick Access 服务: %w", err)
	}
	return owned, func() { _ = owned.Close() }, nil
}
