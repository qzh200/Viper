package app

import (
	"viper/internal/domain"
)

type fakeGaming struct {
	powerMode domain.PowerMode

	gpuMode domain.GPUMode

	brightness uint64

	sensors domain.Sensors

	fans domain.FanState

	backlight domain.BacklightState

	bootSound bool

	winKeyLocked bool

	lcdOverdrive bool

	coolBoost bool

	fanTable uint64

	discreteGPU bool

	overclockGPU bool

	overclockCPU bool

	lighting domain.RGBConfig

	brightnessError error

	monitorError error

	closed bool

	written []domain.RGBConfig

	writtenFans []domain.FanConfig

	writtenPower []domain.PowerMode

	writtenBacklight []int

	writtenTimeout []bool

	writtenBootSound []bool

	writtenWinKey []bool

	writtenOverdrive []bool

	writtenCoolBoost []bool

	writtenGPUMode []domain.GPUMode
}

func newFakeGaming() *fakeGaming {
	return &fakeGaming{
		powerMode:  domain.PowerBalanced,
		gpuMode:    domain.GPUModeHybrid,
		brightness: 50,
	}
}

func (g *fakeGaming) Instance() string { return "gaming\\instance" }

func (g *fakeGaming) Sensors() (domain.Sensors, error) { return g.sensors, nil }

func (g *fakeGaming) PowerMode() (domain.PowerMode, error) { return g.powerMode, nil }

func (g *fakeGaming) SetPowerMode(mode domain.PowerMode) error {
	g.writtenPower = append(g.writtenPower, mode)
	g.powerMode = mode
	return nil
}

func (g *fakeGaming) GPUMode() (domain.GPUMode, error) { return g.gpuMode, nil }

func (g *fakeGaming) SetGPUMode(mode domain.GPUMode) error {
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

func (g *fakeGaming) Lighting() (domain.RGBConfig, error) {
	if g.brightnessError != nil {
		return domain.RGBConfig{}, g.brightnessError
	}
	result := g.lighting
	result.Brightness = g.brightness
	result.ColourReadable = true
	result.EffectReadable = true
	return result, nil
}

func (g *fakeGaming) SetLighting(config domain.RGBConfig) error {
	g.written = append(g.written, config)

	g.lighting = config
	return nil
}

func (g *fakeGaming) ResetLighting() error {

	white := domain.RGBColor{R: 255, G: 255, B: 255}
	config := domain.RGBConfig{
		Effect:     domain.EffectStatic,
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

func (g *fakeGaming) Backlight() (domain.BacklightState, error) { return g.backlight, nil }

func (g *fakeGaming) BacklightBuffer() ([]byte, error) {

	buffer := make([]byte, 16)
	buffer[2] = byte(g.brightness)
	buffer[5] = byte(g.brightness)
	buffer[8] = 3
	buffer[9] = 1
	return buffer, nil
}

func (g *fakeGaming) SetBacklight(brightness int, timeout bool) error {
	g.writtenBacklight = append(g.writtenBacklight, brightness)
	g.writtenTimeout = append(g.writtenTimeout, timeout)

	g.brightness = uint64(brightness)
	g.backlight = domain.BacklightState{
		Brightness: brightness,
		Timeout:    timeout,
	}
	return nil
}

func (g *fakeGaming) Monitor() (domain.MonitorData, error) {
	if g.monitorError != nil {
		return domain.MonitorData{}, g.monitorError
	}
	return domain.MonitorData{
		CPUTemperatureC:    90,
		CPUFanRPM:          2700,
		CPUUsagePercent:    12.5,
		CPUFrequencyMHz:    3400,
		CPUMaxFrequencyMHz: 5000,
		SystemTempC:        63,
		GPUTemperatureC:    58,
		GPUFanRPM:          2705,
		GPUUsagePercent:    39,
		GPUFrequencyMHz:    420,
		GPUMaxFrequencyMHz: 3105,
		RAMTotalMB:         32487,
		RAMUsagePercent:    42.8,
		RAMFrequencyMHz:    5600,
	}, nil
}

func (g *fakeGaming) Fans() (domain.FanState, error) {
	state := g.fans
	state.CoolBoost = state.CPUMode == domain.FanMax && state.GPUMode == domain.FanMax
	return state, nil
}

func (g *fakeGaming) SetFans(config domain.FanConfig) error {
	g.writtenFans = append(g.writtenFans, config)

	g.fans = domain.FanState{
		CPUMode:    config.CPUMode,
		GPUMode:    config.GPUMode,
		CPUPercent: config.CPUPercent,
		GPUPercent: config.GPUPercent,
		CoolBoost:  config.CoolBoost,
	}
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
