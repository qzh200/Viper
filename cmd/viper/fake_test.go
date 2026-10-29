package main

import (
	"path/filepath"
	"testing"

	"viper/internal/app"
	"viper/internal/config"
	"viper/internal/domain"
	"viper/internal/quickaccess"
)

type fakeMachine struct {
	power     domain.PowerMode
	gpu       domain.GPUMode
	lighting  domain.RGBConfig
	backlight domain.BacklightState
	fans      domain.FanState
	winKey    bool
	overdrive bool
	bootSound bool
	monitor   domain.MonitorData

	lightingErr error
}

func newFakeMachine() *fakeMachine {
	return &fakeMachine{
		power:     domain.PowerBalanced,
		gpu:       domain.GPUModeDiscrete,
		lighting:  domain.RGBConfig{Effect: domain.EffectWave, Speed: 3, Direction: 1, Brightness: 5, PerZone: true},
		backlight: domain.BacklightState{Brightness: 5, Timeout: true},
		fans:      domain.FanState{CPUMode: domain.FanAuto, GPUMode: domain.FanAuto, GPUModeReadable: true},
		monitor: domain.MonitorData{
			CPUTemperatureC:     87,
			CPUFanRPM:           2681,
			CPUUsagePercent:     7,
			CPUFrequencyMHz:     2653,
			CPUMaxFrequencyMHz:  5000,
			CPUCoreFrequencyMHz: []int{1671, 3408},
			GPUTemperatureC:     56,
			GPUFanRPM:           2681,
			GPUFrequencyMHz:     600,
			GPUMaxFrequencyMHz:  3105,
			SystemTempC:         61,
			RAMTotalMB:          32488,
			RAMUsagePercent:     43.1,
			RAMFrequencyMHz:     5600,
			WIFIDownloadKbps:    1678.1,
			WIFIUploadKbps:      251,
		},
	}
}

func (m *fakeMachine) Instance() string { return "假机器" }

func (m *fakeMachine) Close() {}

func (m *fakeMachine) Sensors() (domain.Sensors, error) {
	return domain.Sensors{
		CPUTempC:    m.monitor.CPUTemperatureC,
		CPUFanRPM:   m.monitor.CPUFanRPM,
		GPUTempC:    m.monitor.GPUTemperatureC,
		GPUFanRPM:   m.monitor.GPUFanRPM,
		SystemTempC: m.monitor.SystemTempC,
	}, nil
}

func (m *fakeMachine) Monitor() (domain.MonitorData, error) { return m.monitor, nil }

func (m *fakeMachine) PowerMode() (domain.PowerMode, error) { return m.power, nil }

func (m *fakeMachine) SetPowerMode(mode domain.PowerMode) error {
	m.power = mode
	return nil
}

func (m *fakeMachine) GPUMode() (domain.GPUMode, error) { return m.gpu, nil }

func (m *fakeMachine) SetGPUMode(mode domain.GPUMode) error {
	m.gpu = mode
	return nil
}

func (m *fakeMachine) DiscreteGPU() (bool, error) {
	return m.gpu == domain.GPUModeDiscrete, nil
}

func (m *fakeMachine) OverclockCapability() (bool, bool, error) { return false, false, nil }

func (m *fakeMachine) LightingBrightness() (uint64, error) {
	return m.lighting.Brightness, nil
}

func (m *fakeMachine) Backlight() (domain.BacklightState, error) { return m.backlight, nil }

func (m *fakeMachine) Lighting() (domain.RGBConfig, error) {
	if m.lightingErr != nil {
		return domain.RGBConfig{}, m.lightingErr
	}
	return m.lighting, nil
}

func (m *fakeMachine) SetLighting(config domain.RGBConfig) error {
	m.lighting = config
	return nil
}

func (m *fakeMachine) SetBacklight(brightness int, timeout bool) error {
	m.backlight = domain.BacklightState{Brightness: brightness, Timeout: timeout}
	return nil
}

func (m *fakeMachine) ResetLighting() error {
	m.lighting = domain.RGBConfig{Effect: domain.EffectStatic, Speed: 3, Brightness: 100, PerZone: true}
	return nil
}

func (m *fakeMachine) Fans() (domain.FanState, error) { return m.fans, nil }

func (m *fakeMachine) SetFans(config domain.FanConfig) error {
	m.fans = domain.FanState{
		CPUMode:         config.CPUMode,
		GPUMode:         config.GPUMode,
		GPUModeReadable: true,
		CPUPercent:      config.CPUPercent,
		GPUPercent:      config.GPUPercent,
		CoolBoost:       config.CoolBoost,
	}
	return nil
}

func (m *fakeMachine) CoolBoostEnabled() (bool, error) { return m.fans.CoolBoost, nil }

func (m *fakeMachine) CoolBoost(enabled bool) error {
	m.fans.CoolBoost = enabled
	return nil
}

func (m *fakeMachine) WinKeyLocked() (bool, error) { return m.winKey, nil }

func (m *fakeMachine) SetWinKeyLocked(locked bool) error {
	m.winKey = locked
	return nil
}

func (m *fakeMachine) LCDOverdrive() (bool, error) { return m.overdrive, nil }

func (m *fakeMachine) SetLCDOverdrive(enabled bool) error {
	m.overdrive = enabled
	return nil
}

func (m *fakeMachine) BootSound() (bool, error) { return m.bootSound, nil }

func (m *fakeMachine) SetBootSound(enabled bool) error {
	m.bootSound = enabled
	return nil
}

type fakeQuick struct {
	chargeLimit  int
	chargeUSB    bool
	display      quickaccess.DisplayMode
	dark         bool
	transparent  bool
	taskbar      bool
	saver        bool
	saverTimeout int
}

func newFakeQuick() *fakeQuick {
	return &fakeQuick{
		chargeLimit:  30,
		chargeUSB:    true,
		display:      quickaccess.DisplayModeDiscrete,
		saverTimeout: 600,
	}
}

func (q *fakeQuick) USBChargeLimit() (int, error) { return q.chargeLimit, nil }

func (q *fakeQuick) SetUSBChargeLimit(percent int) error {
	q.chargeLimit = percent
	return nil
}

func (q *fakeQuick) USBChargeEnabled() (bool, error) { return q.chargeUSB, nil }

func (q *fakeQuick) SetUSBChargeEnabled(enabled bool) error {
	q.chargeUSB = enabled
	return nil
}

func (q *fakeQuick) DisplayMode() (quickaccess.DisplayMode, error) { return q.display, nil }

func (q *fakeQuick) SetDisplayMode(mode quickaccess.DisplayMode) error {
	q.display = mode
	return nil
}

func (q *fakeQuick) DarkMode() (bool, error) { return q.dark, nil }

func (q *fakeQuick) SetDarkMode(enabled bool) error {
	q.dark = enabled
	return nil
}

func (q *fakeQuick) TransparentEffect() (bool, error) { return q.transparent, nil }

func (q *fakeQuick) SetTransparentEffect(enabled bool) error {
	q.transparent = enabled
	return nil
}

func (q *fakeQuick) AutoHideTaskbar() (bool, error) { return q.taskbar, nil }

func (q *fakeQuick) SetAutoHideTaskbar(enabled bool) error {
	q.taskbar = enabled
	return nil
}

func (q *fakeQuick) ScreenSaver() (bool, error) { return q.saver, nil }

func (q *fakeQuick) SetScreenSaver(enabled bool) error {
	q.saver = enabled
	return nil
}

func (q *fakeQuick) ScreenSaverTimeout() (int, error) { return q.saverTimeout, nil }

func (q *fakeQuick) SetScreenSaverTimeout(seconds int) error {
	q.saverTimeout = seconds
	return nil
}

func (q *fakeQuick) Close() error { return nil }

func installFakes(t *testing.T) (*fakeMachine, *fakeQuick) {
	t.Helper()

	machine := newFakeMachine()
	quick := newFakeQuick()

	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}

	old := newApp
	newApp = func() (*app.App, error) {
		application := app.NewWithQuickAccess(store, quick)
		application.SetBackendOpener(func() (domain.Gaming, string, error) {
			return machine, "测试路线", nil
		})
		return application, nil
	}
	t.Cleanup(func() { newApp = old })

	return machine, quick
}
