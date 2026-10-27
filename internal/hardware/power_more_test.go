package hardware

import (
	"testing"
)

// The performance profile travels in the second byte of the setting, the
// identifier of the setting in the lowest one.

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
