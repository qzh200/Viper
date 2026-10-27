// Package app is the use case layer of Viper.
//
// It joins the hardware backend with the stored settings. The firmware
// reports the fan and the power state but not the lighting state, so the
// lighting is kept in the configuration and only the brightness is read
// back from the hardware.
//
// The settings that do not belong to the gaming hardware, such as the blue
// light shield, are read from the service of the original software.
package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/hardware"
	"viper/internal/nvidia"
	"viper/internal/quickaccess"
)

// App is the running program.
type App struct {
	backend *hardware.Backend
	store   *config.Store
}

// New opens the stored settings.
//
// The connection to the gaming hardware is opened on first use, the
// settings of the original software do not need the rights that the
// hardware needs.
func New() (*App, error) {
	store, err := config.Open()
	if err != nil {
		return nil, err
	}
	return NewWithStore(store), nil
}

// NewWithStore opens the program with the given settings.
//
// It is used by the tests, so that they do not touch the settings of the
// user.
func NewWithStore(store *config.Store) *App {
	return &App{store: store}
}

// Close releases the hardware.
func (a *App) Close() {
	if a.backend != nil {
		a.backend.Close()
	}
}

// hardware opens the connection to the gaming hardware.
//
// The provider refuses the calls without the rights of an administrator,
// so the connection is only made when a hardware setting is used.
func (a *App) hardware() (*hardware.Backend, error) {
	if a.backend != nil {
		return a.backend, nil
	}
	backend, err := hardware.NewBackend()
	if err != nil {
		return nil, err
	}
	a.backend = backend
	return backend, nil
}

// Instance returns the object path of the gaming functions.
func (a *App) Instance() (string, error) {
	backend, err := a.hardware()
	if err != nil {
		return "", err
	}
	return backend.Instance(), nil
}

// SettingsPath returns the location of the settings file.
func (a *App) SettingsPath() string { return a.store.Path() }

// Sensors reads the live temperatures and fan speeds.
func (a *App) Sensors() (hardware.Sensors, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.Sensors{}, err
	}
	return backend.Sensors()
}

// PowerStatus reads the power supply of the machine.
//
// The information comes from the kernel of Windows, so it needs neither the
// gaming provider nor the services of the original software.
func (a *App) PowerStatus() (hardware.PowerStatus, error) {
	return hardware.SystemPowerStatus()
}

// GPUTelemetry reads the state of the graphics card.
//
// The numbers come from the driver of the card. A machine without a card
// that offers them answers with an empty list.
func (a *App) GPUTelemetry() ([]nvidia.Telemetry, error) {
	cards, err := nvidia.Read()
	if err != nil {
		// A machine without the driver of the vendor simply has no
		// telemetry, that is not an error.
		if nvidia.Available() {
			return nil, err
		}
		return nil, nil
	}
	return cards, nil
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

	backend, err := a.hardware()
	if err != nil {
		return result, nil
	}
	brightness, err := backend.LightingBrightness()
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
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetLighting(value); err != nil {
		return err
	}

	stored := toStoredLighting(value)
	return a.store.Update(func(s *config.Settings) {
		s.Lighting = stored
	})
}

// toStoredLighting converts a lighting configuration into its stored form.
//
// The values are limited to the range that the firmware accepts, so that
// the stored settings never claim more than the keyboard can do. The
// limits come from the profile of the original software: the brightness
// runs from zero to one hundred, the speed from one to nine and the
// direction is a switch.
func toStoredLighting(value hardware.RGBConfig) config.Lighting {
	stored := config.Lighting{
		Effect:     uint64(value.Effect),
		Speed:      clamp(value.Speed, 1, 9),
		Direction:  clamp(value.Direction, 0, 1),
		Brightness: clamp(value.Brightness, 0, 100),
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
	backend, err := a.hardware()
	if err != nil {
		return hardware.FanState{}, err
	}
	return backend.Fans()
}

// SetFans applies a fan configuration and remembers it.
func (a *App) SetFans(fans hardware.FanConfig) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetFans(fans); err != nil {
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
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.CoolBoostEnabled()
}

// SetCoolBoost switches the boost fan table.
func (a *App) SetCoolBoost(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.CoolBoost(enabled); err != nil {
		return err
	}
	return a.store.Update(func(s *config.Settings) {
		s.Fans.CoolBoost = enabled
	})
}

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
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetPowerMode(mode); err != nil {
		return err
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.PowerMode = uint64(mode)
	}); err != nil {
		return err
	}

	scene, ok := a.store.Settings().Bindings[mode.String()]
	if !ok {
		return nil
	}
	scenario, ok := a.store.Settings().Scenarios[scene]
	if !ok {
		return nil
	}
	return a.applyScenario(scenario)
}

// Scenarios returns the saved scenes.
func (a *App) Scenarios() map[string]config.Scenario {
	return a.store.Settings().Scenarios
}

// Bindings returns the scenes that are bound to a performance profile.
func (a *App) Bindings() map[string]string {
	return a.store.Settings().Bindings
}

// SaveScenario stores the current lighting and fans under a name.
//
// The fan configuration and the brightness are read from the machine, so
// the scene holds what the hardware really does. The effect and the
// colours cannot be read back, they are taken from the settings of Viper.
func (a *App) SaveScenario(name string) error {
	if name == "" {
		return fmt.Errorf("app: the scene needs a name")
	}

	settings := a.store.Settings()
	scenario := config.Scenario{
		Lighting: settings.Lighting,
		Fans:     settings.Fans,
	}

	if state, err := a.Fans(); err == nil {
		scenario.Fans = config.Fans{
			CPUMode:    uint64(state.CPUMode),
			GPUMode:    uint64(state.GPUMode),
			CPUPercent: state.CPUPercent,
			GPUPercent: state.GPUPercent,
			CoolBoost:  state.CoolBoost,
		}
	}

	if backend, err := a.hardware(); err == nil {
		// The firmware reports a level of zero while the backlight has
		// switched itself off, so a zero is not taken for the level the
		// user has chosen.
		if brightness, err := backend.LightingBrightness(); err == nil && brightness > 0 {
			scenario.Lighting.Brightness = brightness
		}
	}

	return a.store.Update(func(s *config.Settings) {
		s.Scenarios[name] = scenario
	})
}

// DeleteScenario removes a scene and the bindings that point to it.
func (a *App) DeleteScenario(name string) error {
	return a.store.Update(func(s *config.Settings) {
		delete(s.Scenarios, name)
		for profile, scene := range s.Bindings {
			if scene == name {
				delete(s.Bindings, profile)
			}
		}
	})
}

// ApplyScenario applies a scene to the machine.
func (a *App) ApplyScenario(name string) error {
	scenario, ok := a.store.Settings().Scenarios[name]
	if !ok {
		return fmt.Errorf("app: there is no scene called %q", name)
	}
	return a.applyScenario(scenario)
}

// applyScenario writes a scene to the machine and remembers it.
func (a *App) applyScenario(scenario config.Scenario) error {
	lighting := hardware.RGBConfig{
		Effect:     hardware.LightingEffect(scenario.Lighting.Effect),
		Speed:      scenario.Lighting.Speed,
		Direction:  scenario.Lighting.Direction,
		Brightness: scenario.Lighting.Brightness,
		PerZone:    scenario.Lighting.PerZone,
	}
	for i, colour := range scenario.Lighting.Zones {
		lighting.Zones[i] = hardware.RGBColor{R: colour.R, G: colour.G, B: colour.B}
	}
	if err := a.SetLighting(lighting); err != nil {
		return err
	}

	return a.SetFans(hardware.FanConfig{
		CPUMode:    hardware.FanMode(scenario.Fans.CPUMode),
		GPUMode:    hardware.FanMode(scenario.Fans.GPUMode),
		CPUPercent: scenario.Fans.CPUPercent,
		GPUPercent: scenario.Fans.GPUPercent,
		CoolBoost:  scenario.Fans.CoolBoost,
	})
}

// BindScenario ties a performance profile to a scene.
func (a *App) BindScenario(profile, scene string) error {
	if _, ok := a.store.Settings().Scenarios[scene]; !ok {
		return fmt.Errorf("app: there is no scene called %q", scene)
	}
	return a.store.Update(func(s *config.Settings) {
		s.Bindings[profile] = scene
	})
}

// UnbindScenario removes the scene of a performance profile.
func (a *App) UnbindScenario(profile string) error {
	return a.store.Update(func(s *config.Settings) {
		delete(s.Bindings, profile)
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
		return fmt.Errorf("app: read the keyboard brightness: %w", err)
	}
	if err := backend.SetBacklight(int(brightness), enabled); err != nil {
		return err
	}
	return a.store.Update(func(s *config.Settings) {
		s.BacklightTimeout = enabled
	})
}

// LCDOverdrive reports whether the display overdrive is active.
func (a *App) LCDOverdrive() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.LCDOverdrive()
}

// SetLCDOverdrive enables or disables the display overdrive.
func (a *App) SetLCDOverdrive(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	return backend.SetLCDOverdrive(enabled)
}

// WinKeyLocked reports whether the Windows key is locked.
func (a *App) WinKeyLocked() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.WinKeyLocked()
}

// SetWinKeyLocked locks or unlocks the Windows key.
func (a *App) SetWinKeyLocked(locked bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	return backend.SetWinKeyLocked(locked)
}

// DiscreteGPU reports the state of the discrete GPU switch.
func (a *App) DiscreteGPU() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.DiscreteGPU()
}

// OverclockCapability reports which overclocking features the machine has.
func (a *App) OverclockCapability() (gpu bool, cpu bool, err error) {
	backend, err := a.hardware()
	if err != nil {
		return false, false, err
	}
	return backend.OverclockCapability()
}

// StickyKeys reports whether the sticky keys are switched on.
//
// The sticky keys belong to the accessibility settings of the current
// user, they are not part of the gaming hardware.
func (a *App) StickyKeys() (bool, error) { return hardware.StickyKeys() }

// SetStickyKeys switches the sticky keys on or off.
func (a *App) SetStickyKeys(enabled bool) error { return hardware.SetStickyKeys(enabled) }

// GPUMode reads the graphics mode of the machine.
func (a *App) GPUMode() (hardware.GPUMode, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.GPUModeHybrid, err
	}
	return backend.GPUMode()
}

// SetGPUMode selects the graphics mode of the machine.
func (a *App) SetGPUMode(mode hardware.GPUMode) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	return backend.SetGPUMode(mode)
}

// BootSound reports whether the sound at the start of the machine is on.
func (a *App) BootSound() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.BootSound()
}

// SetBootSound switches the sound at the start of the machine on or off.
func (a *App) SetBootSound(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	return backend.SetBootSound(enabled)
}

// BlueLightShield reads the level of the blue light shield.
//
// The setting belongs to the Quick Access service of the original stack,
// it is not part of the gaming hardware provider.
func (a *App) BlueLightShield() (quickaccess.BlueLight, error) {
	client, err := a.quick()
	if err != nil {
		return quickaccess.BlueLightDisabled, err
	}
	defer client.Close()
	return client.BlueLightShield()
}

// SetBlueLightShield selects a level of the blue light shield.
func (a *App) SetBlueLightShield(level quickaccess.BlueLight) error {
	client, err := a.quick()
	if err != nil {
		return err
	}
	defer client.Close()
	return client.SetBlueLightShield(level)
}

// USBChargeLimit reads the charging limit in percent.
func (a *App) USBChargeLimit() (int, error) {
	client, err := a.quick()
	if err != nil {
		return 0, err
	}
	defer client.Close()
	return client.USBChargeLimit()
}

// SetUSBChargeLimit sets the charging limit in percent.
func (a *App) SetUSBChargeLimit(percent int) error {
	client, err := a.quick()
	if err != nil {
		return err
	}
	defer client.Close()
	return client.SetUSBChargeLimit(percent)
}

// USBChargeEnabled reports whether the charging over USB is allowed.
func (a *App) USBChargeEnabled() (bool, error) {
	client, err := a.quick()
	if err != nil {
		return false, err
	}
	defer client.Close()
	return client.USBChargeEnabled()
}

// SetUSBChargeEnabled allows or forbids the charging over USB.
func (a *App) SetUSBChargeEnabled(enabled bool) error {
	client, err := a.quick()
	if err != nil {
		return err
	}
	defer client.Close()
	return client.SetUSBChargeEnabled(enabled)
}

// quick opens a session with the Quick Access service of the original
// stack.
//
// The service owns the settings that do not belong to the gaming
// hardware, such as the blue light shield and the charging over USB. It
// is not needed for the other settings, so the session is only opened
// when one of them is used.
func (a *App) quick() (*quickaccess.Client, error) {
	client, err := quickaccess.Connect(quickaccess.Options{})
	if err != nil {
		return nil, fmt.Errorf("app: the Quick Access service of the original software is not reachable: %w", err)
	}
	return client, nil
}
