// Package guitest 提供界面层测试用的替身：假机器与假配置存储。
//
// 它只被测试引用，不会被编进 Viper 的可执行文件里。
package guitest

import (
	"errors"

	"viper/internal/domain"
	"viper/internal/gui/controller"
)

// ErrHardware 是假机器在出错时返回的原因。
var ErrHardware = errors.New("guitest: 机器没有回应")

// Hardware 是一台听话的假机器：读数由测试给定，写入会记下来。
type Hardware struct {
	// MonitorData 是机器读到的整机遥测。
	MonitorData domain.MonitorData
	// Power 是机器当前的性能档位。
	Power domain.PowerMode
	// FansState 是机器当前的风扇配置与状态。
	FansState domain.FanState
	// LightingConfig 是机器当前的键盘灯效配置。
	LightingConfig domain.RGBConfig
	// BacklightTimeoutState 是键盘背光 30 秒休眠开关。
	BacklightTimeoutState bool
	// GPUModeState 是机器当前的显卡工作模式。
	GPUModeState domain.GPUMode
	// DiscreteGPUState 标记是否处于独显直连。
	DiscreteGPUState bool
	// LCDOverdriveState 标记是否开启屏幕超速。
	LCDOverdriveState bool
	// WinKeyLockedState 标记 Win 键是否被锁定。
	WinKeyLockedState bool
	// BootSoundState 标记开机音效是否开启。
	BootSoundState bool
	// InstanceName 是后端驱动实例的名字。
	InstanceName string
	// RouteName 是连接通道的名字。
	RouteName string
	// Err 让每一次读取都失败，模拟原厂服务没有运行。
	Err error
	// SetErr 让每一次写入失败。
	SetErr error

	// Writes 记下每次写入了什么档位。
	Writes []domain.PowerMode
	// FanWrites 记下每次写入了什么风扇配置。
	FanWrites []domain.FanConfig
	// CoolBoostWrites 记下每次写入了什么强冷设置。
	CoolBoostWrites []bool
	// LightWrites 记下每次写入了什么灯效配置。
	LightWrites []domain.RGBConfig
	// ResetLightCalls 记录重置灯效的调用次数。
	ResetLightCalls int
	// TimeoutWrites 记下每次写入了什么背光超时设置。
	TimeoutWrites []bool
	// GPUModeWrites 记下每次写入了什么显卡模式。
	GPUModeWrites []domain.GPUMode
	// LCDOverdriveWrites 记下每次写入了什么屏幕超速设置。
	LCDOverdriveWrites []bool
	// WinKeyLockedWrites 记下每次写入了什么 Win 键锁定设置。
	WinKeyLockedWrites []bool
	// BootSoundWrites 记下每次写入了什么开机音效设置。
	BootSoundWrites []bool
}

// Interface 返回这个替身，让它可以被塞进控制器。
func (h *Hardware) Interface() controller.Hardware { return h }

// Monitor 读取整机遥测。
func (h *Hardware) Monitor() (domain.MonitorData, error) {
	if h.Err != nil {
		return domain.MonitorData{}, h.Err
	}
	return h.MonitorData, nil
}

// PowerMode 读取当前的性能档位。
func (h *Hardware) PowerMode() (domain.PowerMode, error) {
	if h.Err != nil {
		return domain.PowerBalanced, h.Err
	}
	return h.Power, nil
}

// SetPowerMode 切换性能档位，并记下来。
func (h *Hardware) SetPowerMode(mode domain.PowerMode) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.Writes = append(h.Writes, mode)
	h.Power = mode
	return nil
}

// Route 返回连接通道的名字。
func (h *Hardware) Route() string { return h.RouteName }

// FanState 读取当前风扇状态。
func (h *Hardware) FanState() (domain.FanState, error) {
	if h.Err != nil {
		return domain.FanState{}, h.Err
	}
	return h.FansState, nil
}

// SetFans 应用风扇设置并记下来。
func (h *Hardware) SetFans(config domain.FanConfig) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.FanWrites = append(h.FanWrites, config)
	h.FansState.CPUMode = config.CPUMode
	h.FansState.GPUMode = config.GPUMode
	h.FansState.CPUPercent = config.CPUPercent
	h.FansState.GPUPercent = config.GPUPercent
	h.FansState.CoolBoost = config.CoolBoost
	return nil
}

// SetCoolBoost 设置强冷开关并记下来。
func (h *Hardware) SetCoolBoost(enabled bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.CoolBoostWrites = append(h.CoolBoostWrites, enabled)
	h.FansState.CoolBoost = enabled
	return nil
}

// Lighting 读取当前键盘 RGB 灯效配置。
func (h *Hardware) Lighting() (domain.RGBConfig, error) {
	if h.Err != nil {
		return domain.RGBConfig{}, h.Err
	}
	return h.LightingConfig, nil
}

// SetLighting 应用键盘灯效配置并记下来。
func (h *Hardware) SetLighting(config domain.RGBConfig) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.LightWrites = append(h.LightWrites, config)
	h.LightingConfig = config
	return nil
}

// ResetLighting 恢复出厂默认灯效并记下来。
func (h *Hardware) ResetLighting() error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.ResetLightCalls++
	h.LightingConfig = domain.RGBConfig{
		Effect:     domain.EffectStatic,
		Speed:      3,
		Brightness: 100,
		PerZone:    true,
	}
	for i := range h.LightingConfig.Zones {
		h.LightingConfig.Zones[i] = domain.RGBColor{R: 255, G: 255, B: 255}
	}
	return nil
}

// BacklightTimeout 读取背光超时开关。
func (h *Hardware) BacklightTimeout() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.BacklightTimeoutState, nil
}

// SetBacklightTimeout 设置背光超时开关并记下来。
func (h *Hardware) SetBacklightTimeout(enabled bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.TimeoutWrites = append(h.TimeoutWrites, enabled)
	h.BacklightTimeoutState = enabled
	return nil
}

// GPUMode 查询当前的显卡工作模式。
func (h *Hardware) GPUMode() (domain.GPUMode, error) {
	if h.Err != nil {
		return domain.GPUModeHybrid, h.Err
	}
	return h.GPUModeState, nil
}

// SetGPUMode 切换显卡模式并记下来。
func (h *Hardware) SetGPUMode(mode domain.GPUMode) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.GPUModeWrites = append(h.GPUModeWrites, mode)
	h.GPUModeState = mode
	return nil
}

// DiscreteGPU 查询当前是否工作在纯独显直连模式。
func (h *Hardware) DiscreteGPU() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.DiscreteGPUState, nil
}

// LCDOverdrive 查询屏幕响应时间超速加速功能是否开启。
func (h *Hardware) LCDOverdrive() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.LCDOverdriveState, nil
}

// SetLCDOverdrive 设置屏幕 LCD Overdrive 响应时间加速开关。
func (h *Hardware) SetLCDOverdrive(enabled bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.LCDOverdriveWrites = append(h.LCDOverdriveWrites, enabled)
	h.LCDOverdriveState = enabled
	return nil
}

// WinKeyLocked 查询 Windows 徽标键是否处于锁定禁用状态。
func (h *Hardware) WinKeyLocked() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.WinKeyLockedState, nil
}

// SetWinKeyLocked 启用或禁用 Windows 徽标键锁定。
func (h *Hardware) SetWinKeyLocked(locked bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.WinKeyLockedWrites = append(h.WinKeyLockedWrites, locked)
	h.WinKeyLockedState = locked
	return nil
}

// BootSound 查询开机启动音效是否开启。
func (h *Hardware) BootSound() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.BootSoundState, nil
}

// SetBootSound 设置开机启动音效开关并记下来。
func (h *Hardware) SetBootSound(enabled bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.BootSoundWrites = append(h.BootSoundWrites, enabled)
	h.BootSoundState = enabled
	return nil
}

// Instance 返回后端驱动实例的名字。
func (h *Hardware) Instance() (string, error) {
	if h.Err != nil {
		return "", h.Err
	}
	return h.InstanceName, nil
}

// Store 是一份假的配置存储：只留一个外观模式。
type Store struct {
	// Mode 是存着的外观模式。
	Mode string
	// Err 让保存失败。
	Err error
	// Saved 记下每次保存了什么。
	Saved []string
}

// Interface 返回这个替身，让它可以被塞进控制器。
func (s *Store) Interface() controller.AppearanceStore { return s }

// Theme 返回存着的外观模式。
func (s *Store) Theme() string { return s.Mode }

// SetTheme 保存外观模式，并记下来。
func (s *Store) SetTheme(mode string) error {
	if s.Err != nil {
		return s.Err
	}
	s.Mode = mode
	s.Saved = append(s.Saved, mode)
	return nil
}
