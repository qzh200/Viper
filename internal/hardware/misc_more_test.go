package hardware

import (
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
