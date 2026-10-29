package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/hardware"
)

// PowerMode 查询当前生效的电源性能模式。
func (a *App) PowerMode() (hardware.PowerMode, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.PowerBalanced, err
	}
	return backend.PowerMode()
}

// SetPowerMode 设置电源性能模式，更新本地配置，并自动应用绑定到该模式的场景预设（如果有）。
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

// AvailablePowerModes 返回当前机型支持的电源模式列表。
func (a *App) AvailablePowerModes() []hardware.PowerMode {
	return hardware.AvailablePowerModes()
}
