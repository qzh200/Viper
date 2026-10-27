package app

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"viper/internal/config"
	"viper/internal/hardware"
	"viper/internal/quickaccess"
)

// The tests of this file lock the fixes of the use case layer: the
// connections are opened once, the lighting that is written is the
// lighting that is stored, the settings of the Quick Access service are
// reachable and the stored values carry the enumeration of the hardware
// layer.

func TestTheHardwareIsOpenedOnlyOnce(t *testing.T) {
	// Two calls that run at the same time must not open two connections,
	// the second one would be left behind.
	application := NewWithStore(mustStore(t))

	var mu sync.Mutex
	opened := 0
	machine := newFakeGaming()

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			backend, err := application.connect(func() (hardware.Gaming, error) {
				mu.Lock()
				opened++
				mu.Unlock()
				return machine, nil
			})
			if err != nil {
				t.Errorf("the hardware was not opened: %v", err)
				return
			}
			if backend != hardware.Gaming(machine) {
				t.Error("another machine was opened")
			}
		}()
	}
	wg.Wait()

	if opened != 1 {
		t.Errorf("the hardware was opened %d times, want once", opened)
	}
}

func TestTheHardwareIsNotOpenedAfterTheClose(t *testing.T) {
	application := NewWithStore(mustStore(t))
	application.Close()

	if _, err := application.connect(func() (hardware.Gaming, error) {
		return newFakeGaming(), nil
	}); err == nil {
		t.Error("the hardware was opened although the program is closed")
	}
}

func TestTheCloseReleasesTheHardwareOnlyOnce(t *testing.T) {
	application, machine := newTestAppWith(t)

	application.Close()
	if !machine.closed {
		t.Error("the machine was not released")
	}
	// A second close must not fail and must not release anything twice.
	application.Close()
}

func TestTheStoredProfileIsTheEnumerationOfTheHardwareLayer(t *testing.T) {
	// The settings carry the semantic enumeration of the hardware layer,
	// never the codes that the firmware expects. The mapping has to work
	// in both directions, so that a profile of the hardware layer and the
	// value that is stored for it always mean the same thing.
	cases := []struct {
		mode   hardware.PowerMode
		stored uint64
	}{
		{hardware.PowerQuiet, config.PowerQuiet},
		{hardware.PowerBalanced, config.PowerBalanced},
		{hardware.PowerPerformance, config.PowerPerformance},
		{hardware.PowerTurbo, config.PowerTurbo},
		{hardware.PowerUnknown, config.PowerUnknown},
		{hardware.PowerEco, config.PowerEco},
	}
	for _, c := range cases {
		if got := storedPowerMode(c.mode); got != c.stored {
			t.Errorf("the profile %d is stored as %d, want %d", c.mode, got, c.stored)
		}
		if got := powerMode(c.stored); got != c.mode {
			t.Errorf("the stored profile %d is read as %d, want %d", c.stored, got, c.mode)
		}
	}
}

func TestTheNumbersOfTheSettingsAreTheNumbersOfTheEnumeration(t *testing.T) {
	// The file carries the numbers of the enumeration of the hardware
	// layer. When the hardware layer moves a profile to another number,
	// this test fails and the change is made on purpose instead of
	// silently changing the meaning of the file.
	cases := map[uint64]hardware.PowerMode{
		config.PowerQuiet:       hardware.PowerQuiet,
		config.PowerBalanced:    hardware.PowerBalanced,
		config.PowerPerformance: hardware.PowerPerformance,
		config.PowerTurbo:       hardware.PowerTurbo,
		config.PowerUnknown:     hardware.PowerUnknown,
		config.PowerEco:         hardware.PowerEco,
	}
	for stored, mode := range cases {
		if stored != uint64(mode) {
			t.Errorf("the stored profile %d is not the number of the profile %d of the hardware layer",
				stored, mode)
		}
	}
}

func TestTheStoredFanModeIsTheEnumerationOfTheHardwareLayer(t *testing.T) {
	// The settings carry the semantic enumeration of the hardware layer,
	// never the codes that the firmware expects.
	cases := []struct {
		mode   hardware.FanMode
		stored uint64
	}{
		{hardware.FanAuto, config.FanAuto},
		{hardware.FanMax, config.FanMax},
		{hardware.FanCustom, config.FanCustom},
	}
	for _, c := range cases {
		if got := storedFanMode(c.mode); got != c.stored {
			t.Errorf("the mode %d is stored as %d, want %d", c.mode, got, c.stored)
		}
		if got := fanMode(c.stored); got != c.mode {
			t.Errorf("the stored mode %d is read as %d, want %d", c.stored, got, c.mode)
		}
	}
}

func TestTheProfileThatIsStoredIsTheOneThatIsWritten(t *testing.T) {
	application, machine := newTestAppWith(t)

	if err := application.SetPowerMode(hardware.PowerPerformance); err != nil {
		t.Fatal(err)
	}

	stored := application.store.Settings().PowerMode
	if stored != config.PowerPerformance {
		t.Errorf("the stored profile is %d, want %d", stored, config.PowerPerformance)
	}
	if len(machine.writtenPower) != 1 || machine.writtenPower[0] != hardware.PowerPerformance {
		t.Errorf("the profile was written %v, want the performance one", machine.writtenPower)
	}
}

func TestAProfileThatIsNotKnownIsRefused(t *testing.T) {
	application, machine := newTestAppWith(t)

	if err := application.SetPowerMode(hardware.PowerUnknown); err == nil {
		t.Error("a profile that is not known was applied")
	}
	if len(machine.writtenPower) != 0 {
		t.Errorf("the profile was written %v, want nothing", machine.writtenPower)
	}
}

func TestAFanModeThatIsNotKnownIsRefused(t *testing.T) {
	application, machine := newTestAppWith(t)

	if err := application.SetFans(hardware.FanConfig{
		CPUMode: hardware.FanMode(9),
		GPUMode: hardware.FanAuto,
	}); err == nil {
		t.Error("a mode that is not known was applied")
	}
	if len(machine.writtenFans) != 0 {
		t.Errorf("the fans were written %v, want nothing", machine.writtenFans)
	}
}

func TestTheStoredFanModeIsTheOneThatIsWritten(t *testing.T) {
	application, _ := newTestAppWith(t)

	if err := application.SetFans(hardware.FanConfig{
		CPUMode:    hardware.FanCustom,
		GPUMode:    hardware.FanMax,
		CPUPercent: 60,
	}); err != nil {
		t.Fatal(err)
	}

	stored := application.store.Settings().Fans
	if stored.CPUMode != config.FanCustom {
		t.Errorf("the stored mode of the CPU fan is %d, want %d", stored.CPUMode, config.FanCustom)
	}
	if stored.GPUMode != config.FanMax {
		t.Errorf("the stored mode of the GPU fan is %d, want %d", stored.GPUMode, config.FanMax)
	}
}

func TestTheLightingThatIsWrittenIsTheLightingThatIsStored(t *testing.T) {
	// The firmware and the settings have to carry the same numbers, a
	// value that the firmware does not accept must not reach it.
	application, machine := newTestAppWith(t)

	if err := application.SetLighting(hardware.RGBConfig{
		Effect:     hardware.EffectWave,
		Speed:      99,
		Direction:  5,
		Brightness: 199,
	}); err != nil {
		t.Fatal(err)
	}

	if len(machine.written) != 1 {
		t.Fatalf("the lighting was written %d times, want once", len(machine.written))
	}
	written := machine.written[0]
	if written.Speed != 9 {
		t.Errorf("the written speed is %d, want 9", written.Speed)
	}
	if written.Direction != 1 {
		t.Errorf("the written direction is %d, want 1", written.Direction)
	}
	if written.Brightness != 100 {
		t.Errorf("the written brightness is %d, want 100", written.Brightness)
	}

	stored := application.store.Settings().Lighting
	if stored.Speed != written.Speed || stored.Direction != written.Direction ||
		stored.Brightness != written.Brightness {
		t.Errorf("the stored lighting is %+v, want the written %+v", stored, written)
	}
}

func TestTheLightingFailsWhenTheFirmwareCannotBeRead(t *testing.T) {
	// The caller has to be able to tell a firmware that cannot be read
	// from a brightness of zero.
	application, machine := newTestAppWith(t)
	machine.brightnessError = errors.New("the firmware does not answer")
	application.store.Update(func(s *config.Settings) {
		s.Lighting.Brightness = 65
	})

	if _, err := application.Lighting(); err == nil {
		t.Error("the lighting was read although the firmware does not answer")
	}
}

func TestTheLightingFailsWithoutTheGamingProvider(t *testing.T) {
	application := NewWithStore(mustStore(t))

	if _, err := application.Lighting(); err == nil {
		t.Error("the lighting was read although the gaming provider is not there")
	}
}

func TestTheModeOfTheGraphicsFanIsNotReadFromTheFirmware(t *testing.T) {
	// The provider does not report the group of the graphics fan, the
	// program must not pretend that it read it.
	application, machine := newTestAppWith(t)
	machine.fans = hardware.FanState{CPUMode: hardware.FanCustom, GPUMode: hardware.FanAuto}

	status, err := application.Fans()
	if err != nil {
		t.Fatal(err)
	}
	if status.GPUModeReadable {
		t.Error("the mode of the graphics fan was read from the firmware")
	}
	if status.CPUMode != hardware.FanCustom {
		t.Errorf("the mode of the CPU fan is %d, want the custom one", status.CPUMode)
	}
}

func TestTheModeOfTheGraphicsFanIsTheOneThatWasWritten(t *testing.T) {
	application, _ := newTestAppWith(t)

	if err := application.SetFans(hardware.FanConfig{
		CPUMode: hardware.FanAuto,
		GPUMode: hardware.FanMax,
	}); err != nil {
		t.Fatal(err)
	}

	status, err := application.Fans()
	if err != nil {
		t.Fatal(err)
	}
	if status.GPUMode != hardware.FanMax {
		t.Errorf("the mode of the graphics fan is %d, want the maximum one", status.GPUMode)
	}
	if status.GPUModeReadable {
		t.Error("the mode of the graphics fan was read from the firmware")
	}
}

func TestTheDisplayModeOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)
	service.displayMode = quickaccess.DisplayModeDiscrete

	mode, err := application.DisplayMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != quickaccess.DisplayModeDiscrete {
		t.Errorf("the display mode is %d, want the discrete one", mode)
	}
}

func TestSetTheDisplayModeOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)

	if err := application.SetDisplayMode(quickaccess.DisplayModeAuto); err != nil {
		t.Fatal(err)
	}
	if len(service.writtenDisplayMode) != 1 || service.writtenDisplayMode[0] != quickaccess.DisplayModeAuto {
		t.Errorf("the display mode was written %v, want the automatic one", service.writtenDisplayMode)
	}
}

func TestTheDarkModeOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)
	service.darkMode = true

	enabled, err := application.DarkMode()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the dark mode is switched off")
	}
}

func TestSetTheDarkModeOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)

	if err := application.SetDarkMode(true); err != nil {
		t.Fatal(err)
	}
	if len(service.writtenDarkMode) != 1 || !service.writtenDarkMode[0] {
		t.Errorf("the dark mode was written %v, want it switched on", service.writtenDarkMode)
	}
}

func TestTheTransparencyOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)
	service.transparentEffect = true

	enabled, err := application.TransparentEffect()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the windows are not transparent")
	}
}

func TestSetTheTransparencyOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)

	if err := application.SetTransparentEffect(true); err != nil {
		t.Fatal(err)
	}
	if len(service.writtenTransparent) != 1 || !service.writtenTransparent[0] {
		t.Errorf("the transparency was written %v, want it switched on", service.writtenTransparent)
	}
}

func TestTheHidingOfTheTaskbarOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)
	service.autoHideTaskbar = true

	enabled, err := application.AutoHideTaskbar()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the taskbar does not hide itself")
	}
}

func TestSetTheHidingOfTheTaskbarOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)

	if err := application.SetAutoHideTaskbar(true); err != nil {
		t.Fatal(err)
	}
	if len(service.writtenAutoHide) != 1 || !service.writtenAutoHide[0] {
		t.Errorf("the hiding of the taskbar was written %v, want it switched on", service.writtenAutoHide)
	}
}

func TestTheScreenSaverOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)
	service.screenSaver = true

	enabled, err := application.ScreenSaver()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the screen saver is switched off")
	}
}

func TestSetTheScreenSaverOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)

	if err := application.SetScreenSaver(true); err != nil {
		t.Fatal(err)
	}
	if len(service.writtenScreenSaver) != 1 || !service.writtenScreenSaver[0] {
		t.Errorf("the screen saver was written %v, want it switched on", service.writtenScreenSaver)
	}
}

func TestTheTimeoutOfTheScreenSaverOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)
	service.screenSaverTimeout = 600

	timeout, err := application.ScreenSaverTimeout()
	if err != nil {
		t.Fatal(err)
	}
	if timeout != 600 {
		t.Errorf("the timeout is %d, want 600", timeout)
	}
}

func TestSetTheTimeoutOfTheScreenSaverOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)

	if err := application.SetScreenSaverTimeout(900); err != nil {
		t.Fatal(err)
	}
	if len(service.writtenTimeout) != 1 || service.writtenTimeout[0] != 900 {
		t.Errorf("the timeout was written %v, want 900", service.writtenTimeout)
	}
}

// failingQuickAccess is a service that answers every call with an error.
//
// The program has to report which setting failed, the caller must be able
// to tell it from a service that is simply switched off.
type failingQuickAccess struct {
	*fakeQuickAccess
}

func (q *failingQuickAccess) DarkMode() (bool, error) {
	return false, errors.New("the service does not know the command")
}

func (q *failingQuickAccess) TransparentEffect() (bool, error) {
	return false, errors.New("the service does not know the command")
}

func (q *failingQuickAccess) ScreenSaverTimeout() (int, error) {
	return 0, errors.New("the service does not know the command")
}

func TestTheFailureOfTheServiceIsReported(t *testing.T) {
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := &failingQuickAccess{fakeQuickAccess: &fakeQuickAccess{}}
	application := NewWithQuickAccess(store, service)

	if _, err := application.DarkMode(); err == nil {
		t.Error("the dark mode was read although the service failed")
	}
	if _, err := application.TransparentEffect(); err == nil {
		t.Error("the transparent effect was read although the service failed")
	}
	if _, err := application.ScreenSaverTimeout(); err == nil {
		t.Error("the timeout was read although the service failed")
	}
}

func TestTheSessionOfTheTestIsNotClosed(t *testing.T) {
	// A session that was handed in belongs to the caller, the program
	// must not close it.
	application, service := newAppWithQuick(t)

	if _, err := application.DarkMode(); err != nil {
		t.Fatal(err)
	}
	if service.closed {
		t.Error("the session of the test was closed by the program")
	}
}
