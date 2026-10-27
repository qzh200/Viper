package app

import (
	"viper/internal/hardware"
)

// The fake gaming stands for the machine, so that the tests of the
// program do not need the rights of an administrator and do not touch the
// firmware.

// fakeGaming is a machine that answers from a table.
type fakeGaming struct {
	// powerMode is the performance profile of the machine.
	powerMode hardware.PowerMode
	// gpuMode is the graphics switch of the machine.
	gpuMode hardware.GPUMode
	// brightness is the brightness of the keyboard.
	brightness uint64
	// sensors are the live readings of the machine.
	sensors hardware.Sensors
	// fans are the fan settings of the machine.
	fans hardware.FanState
	// backlight is the state of the keyboard backlight.
	backlight hardware.BacklightState
	// bootSound reports whether the sound at the start of the machine is on.
	bootSound bool
	// winKeyLocked reports whether the Windows key is locked.
	winKeyLocked bool
	// lcdOverdrive reports whether the display overdrive is active.
	lcdOverdrive bool
	// coolBoost reports whether the boost fan table is selected.
	coolBoost bool
	// fanTable is the table of the fans that the machine uses.
	fanTable uint64
	// discreteGPU reports the state of the discrete GPU switch.
	discreteGPU bool
	// overclockGPU reports whether the card can be overclocked.
	overclockGPU bool
	// overclockCPU reports whether the processor can be overclocked.
	overclockCPU bool
	// lighting is the lighting configuration of the machine.
	lighting hardware.RGBConfig
	// brightnessError is the failure of the brightness of the keyboard.
	brightnessError error

	// closed reports whether the connection was released.
	closed bool
	// written records the lighting that was applied.
	written []hardware.RGBConfig
	// writtenFans records the fan configuration that was applied.
	writtenFans []hardware.FanConfig
	// writtenPower records the profile that was applied.
	writtenPower []hardware.PowerMode
	// writtenBacklight records the backlight that was applied.
	writtenBacklight []int
	// writtenTimeout records the automatic switch off that was applied.
	writtenTimeout []bool
	// writtenBootSound records the sound that was applied.
	writtenBootSound []bool
	// writtenWinKey records the lock of the Windows key that was applied.
	writtenWinKey []bool
	// writtenOverdrive records the overdrive that was applied.
	writtenOverdrive []bool
	// writtenCoolBoost records the boost that was applied.
	writtenCoolBoost []bool
	// writtenGPUMode records the graphics switch that was applied.
	writtenGPUMode []hardware.GPUMode
}

// newFakeGaming builds a machine that answers with the defaults.
func newFakeGaming() *fakeGaming {
	return &fakeGaming{
		powerMode:  hardware.PowerBalanced,
		gpuMode:    hardware.GPUModeHybrid,
		brightness: 50,
	}
}

func (g *fakeGaming) Instance() string { return "gaming\\instance" }

func (g *fakeGaming) Sensors() (hardware.Sensors, error) { return g.sensors, nil }

func (g *fakeGaming) PowerMode() (hardware.PowerMode, error) { return g.powerMode, nil }

func (g *fakeGaming) SetPowerMode(mode hardware.PowerMode) error {
	g.writtenPower = append(g.writtenPower, mode)
	g.powerMode = mode
	return nil
}

func (g *fakeGaming) GPUMode() (hardware.GPUMode, error) { return g.gpuMode, nil }

func (g *fakeGaming) SetGPUMode(mode hardware.GPUMode) error {
	g.writtenGPUMode = append(g.writtenGPUMode, mode)
	g.gpuMode = mode
	return nil
}

func (g *fakeGaming) LightingBrightness() (uint64, error) {
	if g.brightnessError != nil {
		return 0, g.brightnessError
	}
	return g.brightness, nil
}

func (g *fakeGaming) Lighting() (hardware.RGBConfig, error) { return g.lighting, nil }

func (g *fakeGaming) SetLighting(config hardware.RGBConfig) error {
	g.written = append(g.written, config)
	return nil
}

func (g *fakeGaming) ResetLighting() error {
	// The machine returns the default appearance, a static white in every
	// zone at full brightness.
	white := hardware.RGBColor{R: 255, G: 255, B: 255}
	config := hardware.RGBConfig{
		Effect:     hardware.EffectStatic,
		Speed:      3,
		Direction:  0,
		Brightness: 100,
		PerZone:    true,
	}
	for i := range config.Zones {
		config.Zones[i] = white
	}
	g.written = append(g.written, config)
	return nil
}

func (g *fakeGaming) Backlight() (hardware.BacklightState, error) { return g.backlight, nil }

// BacklightBuffer reads the raw buffer of the keyboard backlight.
//
// The buffer is built by the hardware layer, so that the stand answers
// in the same shape as the machine does.
func (g *fakeGaming) BacklightBuffer() ([]byte, error) {
	return hardware.NewBacklightBuffer(int(g.brightness)), nil
}

func (g *fakeGaming) SetBacklight(brightness int, timeout bool) error {
	g.writtenBacklight = append(g.writtenBacklight, brightness)
	g.writtenTimeout = append(g.writtenTimeout, timeout)
	return nil
}

func (g *fakeGaming) Fans() (hardware.FanState, error) { return g.fans, nil }

func (g *fakeGaming) SetFans(config hardware.FanConfig) error {
	g.writtenFans = append(g.writtenFans, config)
	return nil
}

func (g *fakeGaming) FanTable() (uint64, error) { return uint64(g.fanTable), nil }

func (g *fakeGaming) CoolBoostEnabled() (bool, error) { return g.coolBoost, nil }

func (g *fakeGaming) CoolBoost(enabled bool) error {
	g.writtenCoolBoost = append(g.writtenCoolBoost, enabled)
	g.coolBoost = enabled
	return nil
}

func (g *fakeGaming) WinKeyLocked() (bool, error) { return g.winKeyLocked, nil }

func (g *fakeGaming) SetWinKeyLocked(locked bool) error {
	g.writtenWinKey = append(g.writtenWinKey, locked)
	g.winKeyLocked = locked
	return nil
}

func (g *fakeGaming) LCDOverdrive() (bool, error) { return g.lcdOverdrive, nil }

func (g *fakeGaming) SetLCDOverdrive(enabled bool) error {
	g.writtenOverdrive = append(g.writtenOverdrive, enabled)
	g.lcdOverdrive = enabled
	return nil
}

func (g *fakeGaming) BootSound() (bool, error) { return g.bootSound, nil }

func (g *fakeGaming) SetBootSound(enabled bool) error {
	g.writtenBootSound = append(g.writtenBootSound, enabled)
	g.bootSound = enabled
	return nil
}

func (g *fakeGaming) DiscreteGPU() (bool, error) { return g.discreteGPU, nil }

func (g *fakeGaming) OverclockCapability() (bool, bool, error) {
	return g.overclockGPU, g.overclockCPU, nil
}

func (g *fakeGaming) Close() { g.closed = true }
