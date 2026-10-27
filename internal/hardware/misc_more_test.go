package hardware

import (
	"errors"
	"testing"
)

// The provider keeps a small table of switches, a setting is addressed by
// its identifier and carries its value in the upper byte of the same word.

func TestMiscSettingValueReadsTheWord(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0100)
	backend := newFakeBackend(provider)

	value, err := backend.MiscSettingValue(uint32(MiscGPUMode))
	if err != nil {
		t.Fatal(err)
	}
	if value != 0x0100 {
		t.Errorf("the setting is %d, want 0x0100", value)
	}
}

func TestMiscSettingValueOfAMachineThatDoesNotAnswer(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if _, err := backend.MiscSettingValue(uint32(MiscGPUMode)); err == nil {
		t.Error("the call was accepted although the provider does not answer")
	}
}

func TestSetMiscSettingValueWritesTheValueAndTheIdentifier(t *testing.T) {
	provider := newFakeProvider()
	// The provider answers with the new value, the program reads the
	// setting back to confirm the change.
	provider.sequence("GetGamingMiscSetting", []uint64{0x0200})
	backend := newFakeBackend(provider)

	if err := backend.SetMiscSettingValue(uint32(MiscGPUMode), 2); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingMiscSetting")
	if !ok {
		t.Fatal("the setting was not written")
	}
	value := (call.Input >> 8) & 0xFF
	if value != 2 {
		t.Errorf("the value is %d, want 2", value)
	}
	if call.Input&0xFF != uint64(MiscGPUMode) {
		t.Errorf("the identifier is %d, want %d", call.Input&0xFF, MiscGPUMode)
	}
}

func TestSetMiscSettingValueReportsAValueThatIsNotTaken(t *testing.T) {
	// The provider keeps the old value, the program has to notice that
	// the change did not happen.
	provider := newFakeProvider()
	provider.sequence("GetGamingMiscSetting", []uint64{0x0100})
	backend := newFakeBackend(provider)

	err := backend.SetMiscSettingValue(uint32(MiscGPUMode), 2)
	if err == nil {
		t.Error("the change was accepted although the provider kept the old value")
	}
}

func TestSetMiscSettingValueReportsTheRefusal(t *testing.T) {
	provider := newFakeProvider()
	provider.fail(errors.New("the provider refuses"))
	backend := newFakeBackend(provider)

	if err := backend.SetMiscSettingValue(uint32(MiscGPUMode), 2); err == nil {
		t.Error("the change was accepted although the provider refuses")
	}
}

func TestTheGPUModeOfTheMachine(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0200)
	backend := newFakeBackend(provider)

	mode, err := backend.GPUMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != GPUModeDiscrete {
		t.Errorf("the graphics mode is %s, want the discrete one", mode)
	}
}

func TestGPUModeOfAMachineThatDoesNotKnowIt(t *testing.T) {
	// A setting that the firmware does not know answers with an error
	// code in the lower byte.
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0202)
	backend := newFakeBackend(provider)

	if _, err := backend.GPUMode(); err == nil {
		t.Error("the mode was accepted although the firmware does not know it")
	}
}

func TestSetGPUMode(t *testing.T) {
	provider := newFakeProvider()
	provider.sequence("GetGamingMiscSetting", []uint64{0x0300})
	backend := newFakeBackend(provider)

	if err := backend.SetGPUMode(GPUModeAuto); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingMiscSetting")
	if !ok {
		t.Fatal("the mode was not written")
	}
	value := (call.Input >> 8) & 0xFF
	if value != uint64(GPUModeAuto) {
		t.Errorf("the value is %d, want %d", value, GPUModeAuto)
	}
}

func TestBootSound(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0100)
	backend := newFakeBackend(provider)

	enabled, err := backend.BootSound()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the sound at the start of the machine is off")
	}
}

func TestBootSoundIsOff(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0000)
	backend := newFakeBackend(provider)

	enabled, err := backend.BootSound()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("the sound at the start of the machine is on")
	}
}

func TestBootSoundOfAMachineThatDoesNotKnowIt(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0102)
	backend := newFakeBackend(provider)

	if _, err := backend.BootSound(); err == nil {
		t.Error("the sound was accepted although the firmware does not know it")
	}
}

func TestSetBootSound(t *testing.T) {
	provider := newFakeProvider()
	provider.sequence("GetGamingMiscSetting", []uint64{0x0100})
	backend := newFakeBackend(provider)

	if err := backend.SetBootSound(true); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingMiscSetting")
	if !ok {
		t.Fatal("the sound was not written")
	}
	value := (call.Input >> 8) & 0xFF
	if value != 1 {
		t.Errorf("the value is %d, want 1", value)
	}
}

func TestSetBootSoundSwitchesItOff(t *testing.T) {
	provider := newFakeProvider()
	provider.sequence("GetGamingMiscSetting", []uint64{0x0000})
	backend := newFakeBackend(provider)

	if err := backend.SetBootSound(false); err != nil {
		t.Fatal(err)
	}

	call, _ := provider.callOf("SetGamingMiscSetting")
	value := (call.Input >> 8) & 0xFF
	if value != 0 {
		t.Errorf("the value is %d, want 0", value)
	}
}
