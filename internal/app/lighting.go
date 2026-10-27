// 键盘的灯效：读、写、复位、背光超时。
package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/hardware"
)

// Lighting returns the keyboard lighting.
//
// 原厂服务读得回分区颜色与灯效，所以这里给的是机器真正的样子。
//
// 读不到的时候这次调用直接失败，调用方必须能分清"机器读不到"和"亮度就是
// 零"，所以这里不再拿设置文件里的旧值顶替。
func (a *App) Lighting() (hardware.RGBConfig, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.RGBConfig{}, err
	}

	result, err := backend.Lighting()
	if err != nil {
		return hardware.RGBConfig{}, fmt.Errorf("app: 读取键盘灯效失败: %w", err)
	}
	return result, nil
}

// StoredLighting returns the keyboard lighting that Viper keeps.
//
// 这是 Viper 自己记着的那一份：上次写进去的灯效与颜色。机器读不到的时候
// 还能用它，但那是 Viper 的记录，不是机器现在的样子。
//
// 这个方法不去问机器，所以机器连不上的时候也能用。
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
// The values are limited to the ranges that the machine accepts before
// they are written, so that the machine and the stored settings always
// carry the same numbers.
//
// 写进机器之前先钳制速度、方向和亮度，写进去的值和记下来的值因此
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
