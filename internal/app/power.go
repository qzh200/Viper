// 性能档位：读、写、这台机器有哪几个。
package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/hardware"
)

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

// AvailablePowerModes lists the profiles this machine supports.
func (a *App) AvailablePowerModes() []hardware.PowerMode {
	return hardware.AvailablePowerModes(hardware.IsPredator())
}
