package hardware

import (
	"testing"
)

// The keyboard lighting is written per zone, but the effect, the speed
// and the direction travel to the whole keyboard in one word.

func TestResetLightingWritesWhiteInEveryZone(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if err := backend.ResetLighting(); err != nil {
		t.Fatal(err)
	}

	calls := provider.callsOf("SetGamingLEDColor")
	if len(calls) != LightingZones {
		t.Fatalf("the zones were written %d times, want %d", len(calls), LightingZones)
	}
	for i, call := range calls {
		colour := call.Input & 0xFFFFFF00
		if colour != 0xFFFFFF00 {
			t.Errorf("the zone %d carries the colour %X, want white", i+1, colour)
		}
	}
}

func TestResetLightingUsesTheStaticPattern(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if err := backend.ResetLighting(); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingLEDBehavior")
	if !ok {
		t.Fatal("the behaviour of the lighting was not written")
	}
	effect := (call.Input >> 16) & 0xFF
	if effect != uint64(EffectStatic) {
		t.Errorf("the pattern is %d, want the static one", effect)
	}
}
