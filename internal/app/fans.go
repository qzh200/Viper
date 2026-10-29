package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/domain"
)

// FanStatus 组合风扇当前硬件状态与 GPU 模式可读性标记。
type FanStatus struct {
	domain.FanState
	GPUModeReadable bool
}

// Fans 读取当前 CPU 与 GPU 风扇的转速模式与百分比。
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

	if !state.GPUModeReadable {
		status.GPUMode = fanMode(a.store.Settings().Fans.GPUMode)
		status.GPUModeReadable = false
	} else {
		status.GPUModeReadable = true
	}
	return status, nil
}

// SetFans 应用新的风扇配置并同步持久化到本地配置文件。
func (a *App) SetFans(fans domain.FanConfig) error {
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

func knownFanMode(mode domain.FanMode) bool {
	switch mode {
	case domain.FanAuto, domain.FanMax, domain.FanCustom:
		return true
	default:
		return false
	}
}

func fanMode(value uint64) domain.FanMode {
	switch value {
	case config.FanMax:
		return domain.FanMax
	case config.FanCustom:
		return domain.FanCustom
	default:
		return domain.FanAuto
	}
}

func storedFanMode(mode domain.FanMode) uint64 {
	switch mode {
	case domain.FanMax:
		return config.FanMax
	case domain.FanCustom:
		return config.FanCustom
	default:
		return config.FanAuto
	}
}

// SetCoolBoost 开启或关闭 CoolBoost 强冷加速。
func (a *App) SetCoolBoost(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.CoolBoost(enabled); err != nil {
		return fmt.Errorf("app: 设置 CoolBoost 失败: %w", err)
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.Fans.CoolBoost = enabled
	}); err != nil {
		return fmt.Errorf("app: 保存风扇加速设置失败: %w", err)
	}
	return nil
}

// CoolBoost 查询 CoolBoost 强冷是否已开启。
func (a *App) CoolBoost() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.CoolBoostEnabled()
}
