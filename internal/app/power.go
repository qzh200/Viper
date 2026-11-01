package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/domain"
)

func (a *App) PowerMode() (domain.PowerMode, error) {
	backend, err := a.hardware()
	if err != nil {
		return domain.PowerBalanced, err
	}
	return backend.PowerMode()
}

func (a *App) SetPowerMode(mode domain.PowerMode) error {
	if mode != domain.PowerQuiet && mode != domain.PowerBalanced &&
		mode != domain.PowerPerformance && mode != domain.PowerTurbo &&
		mode != domain.PowerEco {
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

func powerMode(value uint64) domain.PowerMode {
	switch value {
	case config.PowerQuiet:
		return domain.PowerQuiet
	case config.PowerPerformance:
		return domain.PowerPerformance
	case config.PowerTurbo:
		return domain.PowerTurbo
	case config.PowerUnknown:
		return domain.PowerUnknown
	case config.PowerEco:
		return domain.PowerEco
	default:
		return domain.PowerBalanced
	}
}

func storedPowerMode(mode domain.PowerMode) uint64 {
	switch mode {
	case domain.PowerQuiet:
		return config.PowerQuiet
	case domain.PowerPerformance:
		return config.PowerPerformance
	case domain.PowerTurbo:
		return config.PowerTurbo
	case domain.PowerUnknown:
		return config.PowerUnknown
	case domain.PowerEco:
		return config.PowerEco
	default:
		return config.PowerBalanced
	}
}

func (a *App) AvailablePowerModes() []domain.PowerMode {
	return domain.AvailablePowerModes()
}
