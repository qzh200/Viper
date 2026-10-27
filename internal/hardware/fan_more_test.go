package hardware

import (
	"testing"
)

// The fans are driven through the gaming class of the provider. The
// numbers that travel to the firmware are built here, so they are worth
// pinning down.

func TestFanTable(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingFanTable", "gmOutput", 0x0300)
	backend := newFakeBackend(provider)

	table, err := backend.FanTable()
	if err != nil {
		t.Fatal(err)
	}
	if table != FanTableBoost {
		t.Errorf("the table is %d, want %d", table, FanTableBoost)
	}
}

func TestFanTableOfAMachineThatDoesNotAnswer(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if _, err := backend.FanTable(); err == nil {
		t.Error("the call was accepted although the provider does not answer")
	}
}
