package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/domain"
)

func (a *App) Lighting() (domain.RGBConfig, error) {
	backend, err := a.hardware()
	if err != nil {
		return domain.RGBConfig{}, err
	}

	result, err := backend.Lighting()
	if err != nil {
		return domain.RGBConfig{}, fmt.Errorf("app: 读取键盘灯效失败: %w", err)
	}
	return result, nil
}

func (a *App) StoredLighting() domain.RGBConfig {
	stored := a.store.Settings().Lighting

	result := domain.RGBConfig{
		Effect:     domain.LightingEffect(stored.Effect),
		Speed:      stored.Speed,
		Direction:  stored.Direction,
		Brightness: stored.Brightness,
		PerZone:    stored.PerZone,
	}
	for i, colour := range stored.Zones {
		result.Zones[i] = domain.RGBColor{R: colour.R, G: colour.G, B: colour.B}
	}
	return result
}

func (a *App) SetLighting(value domain.RGBConfig) error {
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

func clampLighting(value domain.RGBConfig) domain.RGBConfig {
	value.Speed = clamp(value.Speed, domain.SpeedMin, domain.SpeedMax)
	value.Direction = clamp(value.Direction, 0, 1)
	value.Brightness = clamp(value.Brightness, domain.BrightnessMin, domain.BrightnessMax)
	return value
}

func toStoredLighting(value domain.RGBConfig) config.Lighting {
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

func clamp(value uint64, low, high uint64) uint64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

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
			Effect:     uint64(domain.EffectStatic),
			Speed:      3,
			Direction:  0,
			Brightness: domain.BrightnessMax,
			PerZone:    false,
		}
		for i := range s.Lighting.Zones {
			s.Lighting.Zones[i] = config.LightingColor{R: 255, G: 255, B: 255}
		}
	}); err != nil {
		return fmt.Errorf("app: 保存键盘灯效设置失败: %w", err)
	}
	return nil
}

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
		return fmt.Errorf("app: 设置键盘背光自动熄灭失败: %w", err)
	}
	return a.store.Update(func(s *config.Settings) {
		s.BacklightTimeout = enabled
	})
}
