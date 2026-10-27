package app

import (
	"path/filepath"
	"testing"

	"viper/internal/config"
	"viper/internal/quickaccess"
)

// The blue light shield and the charging over USB belong to the Quick
// Access service of the original software, not to the gaming hardware.

// fakeQuickAccess is a stand for the service.
type fakeQuickAccess struct {
	// blueLight is the level of the blue light shield.
	blueLight quickaccess.BlueLight
	// chargeLimit is the charging limit in percent.
	chargeLimit int
	// chargeEnabled reports whether the charging over USB is allowed.
	chargeEnabled bool
	// displayMode is the graphics switch of the machine.
	displayMode quickaccess.DisplayMode
	// darkMode reports whether the dark mode of the system is on.
	darkMode bool
	// transparentEffect reports whether the windows are transparent.
	transparentEffect bool
	// autoHideTaskbar reports whether the taskbar hides itself.
	autoHideTaskbar bool
	// screenSaver reports whether the screen saver is switched on.
	screenSaver bool
	// screenSaverTimeout is the idle time before the screen saver starts.
	screenSaverTimeout int

	// writtenBlueLight records the level that was written.
	writtenBlueLight []quickaccess.BlueLight
	// writtenLimit records the limit that was written.
	writtenLimit []int
	// writtenEnabled records the switch that was written.
	writtenEnabled []bool
	// writtenDisplayMode records the graphics switch that was written.
	writtenDisplayMode []quickaccess.DisplayMode
	// writtenDarkMode records the dark mode that was written.
	writtenDarkMode []bool
	// writtenTransparent records the transparency that was written.
	writtenTransparent []bool
	// writtenAutoHide records the hiding of the taskbar that was written.
	writtenAutoHide []bool
	// writtenScreenSaver records the screen saver that was written.
	writtenScreenSaver []bool
	// writtenTimeout records the timeout of the screen saver that was written.
	writtenTimeout []int
	// closed reports whether the session was ended.
	closed bool
}

func (q *fakeQuickAccess) BlueLightShield() (quickaccess.BlueLight, error) {
	return q.blueLight, nil
}

func (q *fakeQuickAccess) SetBlueLightShield(level quickaccess.BlueLight) error {
	q.writtenBlueLight = append(q.writtenBlueLight, level)
	q.blueLight = level
	return nil
}

func (q *fakeQuickAccess) USBChargeLimit() (int, error) {
	return q.chargeLimit, nil
}

func (q *fakeQuickAccess) SetUSBChargeLimit(percent int) error {
	q.writtenLimit = append(q.writtenLimit, percent)
	q.chargeLimit = percent
	return nil
}

func (q *fakeQuickAccess) USBChargeEnabled() (bool, error) {
	return q.chargeEnabled, nil
}

func (q *fakeQuickAccess) SetUSBChargeEnabled(enabled bool) error {
	q.writtenEnabled = append(q.writtenEnabled, enabled)
	q.chargeEnabled = enabled
	return nil
}

func (q *fakeQuickAccess) Close() error {
	q.closed = true
	return nil
}

func (q *fakeQuickAccess) DisplayMode() (quickaccess.DisplayMode, error) {
	return q.displayMode, nil
}

func (q *fakeQuickAccess) SetDisplayMode(mode quickaccess.DisplayMode) error {
	q.writtenDisplayMode = append(q.writtenDisplayMode, mode)
	q.displayMode = mode
	return nil
}

func (q *fakeQuickAccess) DarkMode() (bool, error) {
	return q.darkMode, nil
}

func (q *fakeQuickAccess) SetDarkMode(enabled bool) error {
	q.writtenDarkMode = append(q.writtenDarkMode, enabled)
	q.darkMode = enabled
	return nil
}

func (q *fakeQuickAccess) TransparentEffect() (bool, error) {
	return q.transparentEffect, nil
}

func (q *fakeQuickAccess) SetTransparentEffect(enabled bool) error {
	q.writtenTransparent = append(q.writtenTransparent, enabled)
	q.transparentEffect = enabled
	return nil
}

func (q *fakeQuickAccess) AutoHideTaskbar() (bool, error) {
	return q.autoHideTaskbar, nil
}

func (q *fakeQuickAccess) SetAutoHideTaskbar(enabled bool) error {
	q.writtenAutoHide = append(q.writtenAutoHide, enabled)
	q.autoHideTaskbar = enabled
	return nil
}

func (q *fakeQuickAccess) ScreenSaver() (bool, error) {
	return q.screenSaver, nil
}

func (q *fakeQuickAccess) SetScreenSaver(enabled bool) error {
	q.writtenScreenSaver = append(q.writtenScreenSaver, enabled)
	q.screenSaver = enabled
	return nil
}

func (q *fakeQuickAccess) ScreenSaverTimeout() (int, error) {
	return q.screenSaverTimeout, nil
}

func (q *fakeQuickAccess) SetScreenSaverTimeout(seconds int) error {
	q.writtenTimeout = append(q.writtenTimeout, seconds)
	q.screenSaverTimeout = seconds
	return nil
}

// newAppWithQuick opens the program with a stand for the service.
func newAppWithQuick(t *testing.T) (*App, *fakeQuickAccess) {
	t.Helper()
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := &fakeQuickAccess{}
	return NewWithQuickAccess(store, service), service
}

func TestTheBlueLightShieldOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)
	service.blueLight = quickaccess.BlueLightHigh

	level, err := application.BlueLightShield()
	if err != nil {
		t.Fatal(err)
	}
	if level != quickaccess.BlueLightHigh {
		t.Errorf("the level is %d, want %d", level, quickaccess.BlueLightHigh)
	}
}

func TestSetTheBlueLightShieldOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)

	if err := application.SetBlueLightShield(quickaccess.BlueLightMedium); err != nil {
		t.Fatal(err)
	}
	if len(service.writtenBlueLight) != 1 || service.writtenBlueLight[0] != quickaccess.BlueLightMedium {
		t.Errorf("the level was written %v, want the medium one", service.writtenBlueLight)
	}
}

func TestTheChargingLimitOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)
	service.chargeLimit = 20

	limit, err := application.USBChargeLimit()
	if err != nil {
		t.Fatal(err)
	}
	if limit != 20 {
		t.Errorf("the limit is %d, want 20", limit)
	}
}

func TestSetTheChargingLimitOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)

	if err := application.SetUSBChargeLimit(30); err != nil {
		t.Fatal(err)
	}
	if len(service.writtenLimit) != 1 || service.writtenLimit[0] != 30 {
		t.Errorf("the limit was written %v, want 30", service.writtenLimit)
	}
}

func TestTheChargingOverUSBOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)
	service.chargeEnabled = true

	enabled, err := application.USBChargeEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the charging over USB is not allowed")
	}
}

func TestSetTheChargingOverUSBOfTheService(t *testing.T) {
	application, service := newAppWithQuick(t)

	if err := application.SetUSBChargeEnabled(true); err != nil {
		t.Fatal(err)
	}
	if len(service.writtenEnabled) != 1 || !service.writtenEnabled[0] {
		t.Errorf("the switch was written %v, want it allowed", service.writtenEnabled)
	}
}

func TestTheSettingsOfTheServiceWithoutTheService(t *testing.T) {
	// The service of the original software is not running on every
	// machine, the program has to say that it cannot answer. The session
	// is pointed at a port that nothing listens on.
	application := NewWithStore(mustStore(t))
	application.SetQuickAccessOptions(quickaccess.Options{Address: "127.0.0.1:1"})

	if _, err := application.BlueLightShield(); err == nil {
		t.Error("the level was read although the service is not there")
	}
	if err := application.SetBlueLightShield(quickaccess.BlueLightLow); err == nil {
		t.Error("the level was written although the service is not there")
	}
	if _, err := application.USBChargeLimit(); err == nil {
		t.Error("the limit was read although the service is not there")
	}
	if err := application.SetUSBChargeLimit(10); err == nil {
		t.Error("the limit was written although the service is not there")
	}
	if _, err := application.USBChargeEnabled(); err == nil {
		t.Error("the switch was read although the service is not there")
	}
	if err := application.SetUSBChargeEnabled(true); err == nil {
		t.Error("the switch was written although the service is not there")
	}
}
