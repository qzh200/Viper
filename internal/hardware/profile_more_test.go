package hardware

import (
	"testing"
)

// The profile word carries several switches at once, they are told apart
// by the bit position that is set.

func TestLCDOverdrive(t *testing.T) {
	// The flag of the overdrive sits in the highest byte.
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", uint64(1)<<48)
	backend := newFakeBackend(provider)

	enabled, err := backend.LCDOverdrive()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the overdrive is off")
	}
}

func TestLCDOverdriveIsOff(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0)
	backend := newFakeBackend(provider)

	enabled, err := backend.LCDOverdrive()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("the overdrive is on")
	}
}

func TestLCDOverdriveOfAMachineThatReportsAnError(t *testing.T) {
	// The lower byte carries a status, a non zero one means that the
	// setting is not defined.
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", uint64(1)<<48|0x02)
	backend := newFakeBackend(provider)

	enabled, err := backend.LCDOverdrive()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("the overdrive is on although the setting is not defined")
	}
}

func TestSetLCDOverdriveWritesTheSwitchAndTheFlag(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", uint64(1)<<48)
	backend := newFakeBackend(provider)

	if err := backend.SetLCDOverdrive(true); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingProfile")
	if !ok {
		t.Fatal("the overdrive was not written")
	}
	if call.Input&0xFF != uint64(profileLCDOverdrive) {
		t.Errorf("the switch is %d, want %d", call.Input&0xFF, profileLCDOverdrive)
	}
	if call.Input&profileOverdriveFlag == 0 {
		t.Error("the flag of the overdrive was not set")
	}
}

func TestSetLCDOverdriveSwitchesItOff(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0)
	backend := newFakeBackend(provider)

	if err := backend.SetLCDOverdrive(false); err != nil {
		t.Fatal(err)
	}

	call, _ := provider.callOf("SetGamingProfile")
	if call.Input&profileOverdriveFlag != 0 {
		t.Error("the flag of the overdrive was set although it was switched off")
	}
}

func TestSetLCDOverdriveReportsAChangeThatIsNotTaken(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0)
	backend := newFakeBackend(provider)

	if err := backend.SetLCDOverdrive(true); err == nil {
		t.Error("the change was accepted although the provider kept the old state")
	}
}

func TestWinKeyLocked(t *testing.T) {
	// The lock of the Windows key sits in the fourth byte.
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0x01000000)
	backend := newFakeBackend(provider)

	locked, err := backend.WinKeyLocked()
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Error("the Windows key is not locked")
	}
}

func TestWinKeyIsNotLocked(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0)
	backend := newFakeBackend(provider)

	locked, err := backend.WinKeyLocked()
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Error("the Windows key is locked")
	}
}

func TestSetWinKeyLockedWritesTheSwitchAndTheLock(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0x01000000)
	backend := newFakeBackend(provider)

	if err := backend.SetWinKeyLocked(true); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingProfile")
	if !ok {
		t.Fatal("the lock was not written")
	}
	if call.Input&0xFF != uint64(profileWinKeyLock) {
		t.Errorf("the switch is %d, want %d", call.Input&0xFF, profileWinKeyLock)
	}
	if (call.Input>>24)&0xFF != 1 {
		t.Errorf("the lock is %d, want 1", (call.Input>>24)&0xFF)
	}
}

func TestSetWinKeyUnlocksTheKey(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0)
	backend := newFakeBackend(provider)

	if err := backend.SetWinKeyLocked(false); err != nil {
		t.Fatal(err)
	}

	call, _ := provider.callOf("SetGamingProfile")
	if (call.Input>>24)&0xFF != 0 {
		t.Errorf("the lock is %d, want 0", (call.Input>>24)&0xFF)
	}
}

func TestSetWinKeyLockedReportsAChangeThatIsNotTaken(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0)
	backend := newFakeBackend(provider)

	if err := backend.SetWinKeyLocked(true); err == nil {
		t.Error("the change was accepted although the provider kept the old state")
	}
}

func TestCoolBoost(t *testing.T) {
	// CoolBoost lives on the generic action class rather than on the
	// gaming class of the vendor.
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if err := backend.CoolBoost(true); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetFunction")
	if !ok {
		t.Fatal("the boost was not written")
	}
	if call.Class != ClassAction {
		t.Errorf("the boost was written to the class %q, want %q", call.Class, ClassAction)
	}
	if call.Input&0xFFFF != 7 {
		t.Errorf("the switch is %d, want 7", call.Input&0xFFFF)
	}
	if call.Input&(1<<16) == 0 {
		t.Error("the flag of the boost was not set")
	}
}

func TestCoolBoostSwitchesItOff(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if err := backend.CoolBoost(false); err != nil {
		t.Fatal(err)
	}

	call, _ := provider.callOf("SetFunction")
	if call.Input&(1<<16) != 0 {
		t.Error("the flag of the boost was set although it was switched off")
	}
}

func TestCoolBoostEnabled(t *testing.T) {
	provider := newFakeProvider()
	// The action class answers with the state in the second byte.
	provider.output("GetFunction", "uiOutput", 0x0100)
	backend := newFakeBackend(provider)

	enabled, err := backend.CoolBoostEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the boost is off")
	}
}

func TestCoolBoostFallsBackToTheFanTable(t *testing.T) {
	// The action class does not answer on every machine, the table of the
	// fans carries the state as well.
	provider := newFakeProvider()
	provider.output("GetFunction", "uiOutput", 0)
	provider.output("GetGamingFanTable", "gmOutput", 0x0300)
	backend := newFakeBackend(provider)

	enabled, err := backend.CoolBoostEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the boost is off")
	}
}

func TestCoolBoostIsOff(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetFunction", "uiOutput", 0)
	// The action class does not answer, the table of the fans carries the
	// state. The normal table means that the boost is off.
	provider.output("GetGamingFanTable", "gmOutput", 0x0100)
	backend := newFakeBackend(provider)

	enabled, err := backend.CoolBoostEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("the boost is on")
	}
}

func TestSensors(t *testing.T) {
	// The query is encoded as the category in the upper byte and the
	// index in the lower one. The reading sits in the second byte.
	provider := newFakeProvider()
	provider.sequence("GetGamingSysInfo", []uint64{
		0x3200, // the temperature of the processor
		0x0A00, // the speed of the fan of the processor
		0x2800, // the temperature of the system
		0x0B00, // the speed of the fan of the graphics card
		0x3C00, // the temperature of the graphics card
	})
	backend := newFakeBackend(provider)

	sensors, err := backend.Sensors()
	if err != nil {
		t.Fatal(err)
	}
	if sensors.CPUTempC != 50 {
		t.Errorf("the processor runs at %d degrees, want 50", sensors.CPUTempC)
	}
	if sensors.CPUFanRPM != 10 {
		t.Errorf("the fan of the processor runs at %d revolutions, want 10", sensors.CPUFanRPM)
	}
	if sensors.SystemTempC != 40 {
		t.Errorf("the system runs at %d degrees, want 40", sensors.SystemTempC)
	}
	if sensors.GPUFanRPM != 11 {
		t.Errorf("the fan of the graphics card runs at %d revolutions, want 11", sensors.GPUFanRPM)
	}
	if sensors.GPUTempC != 60 {
		t.Errorf("the graphics card runs at %d degrees, want 60", sensors.GPUTempC)
	}
}

func TestSensorsOfAMachineThatDoesNotAnswer(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if _, err := backend.Sensors(); err == nil {
		t.Error("the readings were accepted although the provider does not answer")
	}
}

func TestTheInstanceOfTheBackend(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if got := backend.Instance(); got != "gaming\\instance" {
		t.Errorf("the instance is %q, want the path of the gaming class", got)
	}
}

func TestTheBackendFallsBackToTheGamingInstance(t *testing.T) {
	provider := newFakeProvider()
	backend := NewBackendWith(provider, "gaming\\instance", "")

	if got := backend.Instance(); got != "gaming\\instance" {
		t.Errorf("the instance is %q, want the path of the gaming class", got)
	}
}
