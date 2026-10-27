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

	// writtenBlueLight records the level that was written.
	writtenBlueLight []quickaccess.BlueLight
	// writtenLimit records the limit that was written.
	writtenLimit []int
	// writtenEnabled records the switch that was written.
	writtenEnabled []bool
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
	application.quickOptions = quickaccess.Options{Address: "127.0.0.1:1"}

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
