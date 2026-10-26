// Package app is the use case layer of Viper.
//
// It joins the hardware backend with the stored settings. The firmware
// reports the fan and the power state but not the lighting state, so the
// lighting is kept in the configuration and only the brightness is read
// back from the hardware.
package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/hardware"
)

// App is the running program.
type App struct {
	backend *hardware.Backend
	store   *config.Store
}

// New opens the hardware and the stored settings.
func New() (*App, error) {
	backend, err := hardware.NewBackend()
	if err != nil {
		return nil, err
	}
	store, err := config.Open()
	if err != nil {
		backend.Close()
		return nil, err
	}
	return &App{backend: backend, store: store}, nil
}

// Close releases the hardware.
func (a *App) Close() { a.backend.Close() }

// Instance returns the object path of the gaming functions.
func (a *App) Instance() string { return a.backend.Instance() }

// SettingsPath returns the location of the settings file.
func (a *App) SettingsPath() string { return a.store.Path() }

// Sensors reads the live temperatures and fan speeds.
func (a *App) Sensors() (hardware.Sensors, error) {
	return a.backend.Sensors()
}

// Lighting returns the keyboard lighting.
//
// The brightness comes from the firmware, the effect and the colours from
// the stored settings.
func (a *App) Lighting() (hardware.RGBConfig, error) {
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

	brightness, err := a.backend.LightingBrightness()
	if err != nil {
		return result, nil
	}
	result.Brightness = brightness

	// The firmware reports the brightness only.
	result.ColourReadable = false
	result.EffectReadable = false
	return result, nil
}

// SetLighting applies a keyboard lighting configuration.
func (a *App) SetLighting(value hardware.RGBConfig) error {
	if err := a.backend.SetLighting(value); err != nil {
		return err
	}

	stored := toStoredLighting(value)
	return a.store.Update(func(s *config.Settings) {
		s.Lighting = stored
	})
}

// toStoredLighting converts a lighting configuration into its stored form.
func toStoredLighting(value hardware.RGBConfig) config.Lighting {
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

// ResetLighting restores the default keyboard lighting.
func (a *App) ResetLighting() error {
	if err := a.backend.ResetLighting(); err != nil {
		return err
	}
	return a.store.Update(func(s *config.Settings) {
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
	})
}

// Fans returns the fan configuration.
func (a *App) Fans() (hardware.FanState, error) {
	return a.backend.Fans()
}

// SetFans applies a fan configuration and remembers it.
func (a *App) SetFans(fans hardware.FanConfig) error {
	if err := a.backend.SetFans(fans); err != nil {
		return err
	}
	return a.store.Update(func(s *config.Settings) {
		s.Fans = config.Fans{
			CPUMode:    uint64(fans.CPUMode),
			GPUMode:    uint64(fans.GPUMode),
			CPUPercent: fans.CPUPercent,
			GPUPercent: fans.GPUPercent,
			CoolBoost:  fans.CoolBoost,
		}
	})
}

// CoolBoost reports whether the boost fan table is selected.
func (a *App) CoolBoost() (bool, error) {
	return a.backend.CoolBoostEnabled()
}

// SetCoolBoost switches the boost fan table.
func (a *App) SetCoolBoost(enabled bool) error {
	if err := a.backend.CoolBoost(enabled); err != nil {
		return err
	}
	return a.store.Update(func(s *config.Settings) {
		s.Fans.CoolBoost = enabled
	})
}

// PowerMode returns the performance profile.
func (a *App) PowerMode() (hardware.PowerMode, error) {
	return a.backend.PowerMode()
}

// SetPowerMode applies a performance profile and remembers it.
func (a *App) SetPowerMode(mode hardware.PowerMode) error {
	if err := a.backend.SetPowerMode(mode); err != nil {
		return err
	}
	return a.store.Update(func(s *config.Settings) {
		s.PowerMode = uint64(mode)
	})
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
	backlight, err := a.backend.Backlight()
	if err != nil {
		return false, err
	}
	return backlight.Timeout, nil
}

// SetBacklightTimeout switches the automatic switch off on its own.
func (a *App) SetBacklightTimeout(enabled bool) error {
	brightness, err := a.backend.LightingBrightness()
	if err != nil {
		return fmt.Errorf("app: read the keyboard brightness: %w", err)
	}
	if err := a.backend.SetBacklight(int(brightness), enabled); err != nil {
		return err
	}
	return a.store.Update(func(s *config.Settings) {
		s.BacklightTimeout = enabled
	})
}

// LCDOverdrive reports whether the display overdrive is active.
func (a *App) LCDOverdrive() (bool, error) { return a.backend.LCDOverdrive() }

// SetLCDOverdrive enables or disables the display overdrive.
func (a *App) SetLCDOverdrive(enabled bool) error { return a.backend.SetLCDOverdrive(enabled) }

// WinKeyLocked reports whether the Windows key is locked.
func (a *App) WinKeyLocked() (bool, error) { return a.backend.WinKeyLocked() }

// SetWinKeyLocked locks or unlocks the Windows key.
func (a *App) SetWinKeyLocked(locked bool) error { return a.backend.SetWinKeyLocked(locked) }

// DiscreteGPU reports the state of the discrete GPU switch.
func (a *App) DiscreteGPU() (bool, error) { return a.backend.DiscreteGPU() }

// OverclockCapability reports which overclocking features the machine has.
func (a *App) OverclockCapability() (gpu bool, cpu bool, err error) {
	return a.backend.OverclockCapability()
}

// StickyKeys reports whether the sticky keys are switched on.
func (a *App) StickyKeys() (bool, error) { return hardware.StickyKeys() }

// SetStickyKeys switches the sticky keys on or off.
func (a *App) SetStickyKeys(enabled bool) error { return hardware.SetStickyKeys(enabled) }

// GPUMode reads the graphics mode of the machine.
func (a *App) GPUMode() (hardware.GPUMode, error) { return a.backend.GPUMode() }

// SetGPUMode selects the graphics mode of the machine.
func (a *App) SetGPUMode(mode hardware.GPUMode) error { return a.backend.SetGPUMode(mode) }

// BluelightShield reads the level of the blue light shield.
//
// The second result is false when the firmware does not know the setting.
func (a *App) BluelightShield() (uint64, bool, error) { return a.backend.BluelightShield() }

// SetBluelightShield selects a level of the blue light shield.
func (a *App) SetBluelightShield(level uint64) error { return a.backend.SetBluelightShield(level) }

// USBChargeLimit reads the charging limit in percent.
//
// The second result is false when the firmware does not know the setting.
func (a *App) USBChargeLimit() (uint64, bool, error) { return a.backend.USBChargeLimit() }

// SetUSBChargeLimit sets the charging limit in percent.
func (a *App) SetUSBChargeLimit(percent uint64) error { return a.backend.SetUSBChargeLimit(percent) }

// USBChargeEnabled reports whether the charging over USB is allowed.
//
// The second result is false when the firmware does not know the setting.
func (a *App) USBChargeEnabled() (bool, bool, error) { return a.backend.USBChargeEnabled() }

// SetUSBChargeEnabled allows or forbids the charging over USB.
func (a *App) SetUSBChargeEnabled(enabled bool) error { return a.backend.SetUSBChargeEnabled(enabled) }
