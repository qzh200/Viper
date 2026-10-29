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
