package service

import (
	"viper/internal/hardware"
)

// The fake gaming stands for the machine, so that the tests of the
// service do not need the rights of an administrator and do not touch the
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

	// written records the lighting that was applied.
	written []hardware.RGBConfig
	// writtenPower records the profile that was applied.
	writtenPower []hardware.PowerMode
	// writtenGPUMode records the graphics switch that was applied.
	writtenGPUMode []hardware.GPUMode
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

func (g *fakeGaming) LightingBrightness() (uint64, error) { return g.brightness, nil }

func (g *fakeGaming) Lighting() (hardware.RGBConfig, error) { return hardware.RGBConfig{}, nil }

func (g *fakeGaming) SetLighting(config hardware.RGBConfig) error {
	g.written = append(g.written, config)
	return nil
}

func (g *fakeGaming) ResetLighting() error { return nil }

func (g *fakeGaming) Backlight() (hardware.BacklightState, error) { return g.backlight, nil }

// BacklightBuffer reads the raw buffer of the keyboard backlight.
//
// The level sits in byte two and in byte five, byte eight holds the flags
// and byte nine has to be one, otherwise the firmware drops the buffer.
func (g *fakeGaming) BacklightBuffer() ([]byte, error) {
	buffer := make([]byte, 16)
	level := uint16(g.brightness * 256 / 100)
	buffer[2] = byte(level)
	buffer[3] = byte(level >> 8)
	buffer[5] = byte(level)
	buffer[6] = byte(level >> 8)
	buffer[9] = 1
	return buffer, nil
}

func (g *fakeGaming) SetBacklight(brightness int, timeout bool) error {
	g.writtenBacklight = append(g.writtenBacklight, brightness)
	g.writtenTimeout = append(g.writtenTimeout, timeout)
	return nil
}

func (g *fakeGaming) Fans() (hardware.FanState, error) { return g.fans, nil }

func (g *fakeGaming) SetFans(config hardware.FanConfig) error { return nil }

func (g *fakeGaming) CoolBoostEnabled() (bool, error) { return false, nil }

func (g *fakeGaming) CoolBoost(enabled bool) error { return nil }

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

func (g *fakeGaming) DiscreteGPU() (bool, error) { return false, nil }

func (g *fakeGaming) OverclockCapability() (bool, bool, error) { return false, false, nil }

func (g *fakeGaming) Close() {}
