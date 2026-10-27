// 风扇：读、写、加速。
package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/hardware"
)

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
