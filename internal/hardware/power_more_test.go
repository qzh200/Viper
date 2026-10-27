package hardware

import (
	"testing"
)

// The performance profile travels in the second byte of the setting, the
// identifier of the setting in the lowest one.

func TestPowerModeReadsTheProfile(t *testing.T) {
	provider := newFakeProvider()
	// The profile sits in the second byte, five is the turbo profile.
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0500)
	backend := newFakeBackend(provider)

	mode, err := backend.PowerMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != PowerTurbo {
		t.Errorf("the profile is %s, want the turbo one", mode)
	}
}

func TestPowerModeOfAProfileThatIsNotThere(t *testing.T) {
	provider := newFakeProvider()
	// Six is the profile that saves the power.
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0600)
	backend := newFakeBackend(provider)

	mode, err := backend.PowerMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != PowerEco {
		t.Errorf("the profile is %s, want the eco one", mode)
	}
}

func TestSetPowerModeWritesTheProfileAndTheIdentifier(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0400)
	backend := newFakeBackend(provider)

	if err := backend.SetPowerMode(PowerPerformance); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingMiscSetting")
	if !ok {
		t.Fatal("the profile was not written")
	}
	// The value travels in the upper byte, the identifier in the lower
	// one.
	value := (call.Input >> 8) & 0xFF
	if value != 4 {
		t.Errorf("the profile is %d, want 4", value)
	}
	if call.Input&0xFF != uint64(MiscOperationMode) {
		t.Errorf("the identifier is %d, want %d", call.Input&0xFF, MiscOperationMode)
	}
}

func TestSetPowerModeReportsAProfileThatIsNotTaken(t *testing.T) {
	// The provider answers with the value it kept, the program has to
	// notice that the change did not happen.
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0100)
	backend := newFakeBackend(provider)

	err := backend.SetPowerMode(PowerTurbo)
	if err == nil {
		t.Error("the change was accepted although the provider kept the old profile")
	}
}

func TestDiscreteGPU(t *testing.T) {
	// The state of the switch sits in the high byte of the setting.
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0300)
	backend := newFakeBackend(provider)

	enabled, err := backend.DiscreteGPU()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the graphics switch is off")
	}
}

func TestDiscreteGPUOfAMachineWithoutTheSwitch(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0000)
	backend := newFakeBackend(provider)

	enabled, err := backend.DiscreteGPU()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("the graphics switch is on")
	}
}

func TestOverclockCapability(t *testing.T) {
	// The provider answers with a non zero value when the feature is
	// there. The setting in the table is the one of the processor, the
	// firmware does not report an overclocking capability of the graphics
	// card, so that one is always reported as missing.
	provider := newFakeProvider()
	provider.sequence("GetGamingMiscSetting", []uint64{0x0100, 0x0100})
	backend := newFakeBackend(provider)

	gpu, cpu, err := backend.OverclockCapability()
	if err != nil {
		t.Fatal(err)
	}
	if gpu {
		t.Error("the overclocking of the graphics card is reported, want it to be missing")
	}
	if !cpu {
		t.Error("the overclocking of the processor is not reported")
	}
}

func TestOverclockCapabilityOfAMachineWithoutIt(t *testing.T) {
	provider := newFakeProvider()
	provider.sequence("GetGamingMiscSetting", []uint64{0x0000, 0x0000})
	backend := newFakeBackend(provider)

	gpu, cpu, err := backend.OverclockCapability()
	if err != nil {
		t.Fatal(err)
	}
	if gpu || cpu {
		t.Errorf("the machine reports the overclocking of the card as %v and of the processor as %v", gpu, cpu)
	}
}

func TestOverclockCapabilityOfAnUnknownValue(t *testing.T) {
	// The provider answers with 0xFF when it does not know the setting.
	provider := newFakeProvider()
	provider.sequence("GetGamingMiscSetting", []uint64{0xFF00, 0xFF00})
	backend := newFakeBackend(provider)

	gpu, cpu, err := backend.OverclockCapability()
	if err != nil {
		t.Fatal(err)
	}
	if gpu || cpu {
		t.Errorf("the machine reports the overclocking of the card as %v and of the processor as %v", gpu, cpu)
	}
}
