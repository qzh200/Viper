package hardware

import (
	"errors"
	"testing"
)

// The keyboard lighting is written per zone, but the effect, the speed
// and the direction travel to the whole keyboard in one word.

func TestSetLightingWritesEveryZone(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	config := RGBConfig{
		Effect:     EffectWave,
		Speed:      5,
		Direction:  1,
		Brightness: 80,
		PerZone:    true,
	}
	config.Zones[0] = RGBColor{R: 199, G: 0, B: 135}
	config.Zones[1] = RGBColor{R: 0, G: 174, B: 199}

	if err := backend.SetLighting(config); err != nil {
		t.Fatal(err)
	}

	calls := provider.callsOf("SetGamingLEDColor")
	if len(calls) != LightingZones {
		t.Fatalf("the zones were written %d times, want %d", len(calls), LightingZones)
	}
	// The colour of a zone travels in the second byte, the number of the
	// zone in the lowest one. The zones are counted from one.
	for i, call := range calls {
		zone := call.Input & 0xFF
		if zone != uint64(i+1) {
			t.Errorf("the zone %d was written as %d", i, zone)
		}
	}
}

func TestSetLightingWritesTheSameColourInEveryZoneWithoutThePerZoneFlag(t *testing.T) {
	// Without the per zone flag every zone carries the colour of the
	// first one.
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	config := RGBConfig{PerZone: false}
	config.Zones[0] = RGBColor{R: 199, G: 0, B: 135}
	config.Zones[1] = RGBColor{R: 0, G: 174, B: 199}

	if err := backend.SetLighting(config); err != nil {
		t.Fatal(err)
	}

	calls := provider.callsOf("SetGamingLEDColor")
	if len(calls) != LightingZones {
		t.Fatalf("the zones were written %d times, want %d", len(calls), LightingZones)
	}
	first := calls[0].Input & 0xFFFFFF00
	for i, call := range calls {
		if call.Input&0xFFFFFF00 != first {
			t.Errorf("the zone %d carries the colour %X, want %X", i+1, call.Input&0xFFFFFF00, first)
		}
	}
}

func TestSetLightingWritesTheBehaviour(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	err := backend.SetLighting(RGBConfig{Effect: EffectMeteor, Speed: 7, Direction: 1})
	if err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingLEDBehavior")
	if !ok {
		t.Fatal("the behaviour of the lighting was not written")
	}
	// The pattern travels in the third byte, the speed in the second one
	// and the direction in the lowest one.
	effect := (call.Input >> 16) & 0xFF
	speed := (call.Input >> 8) & 0xFF
	direction := call.Input & 0xFF
	if effect != uint64(EffectMeteor) {
		t.Errorf("the pattern is %d, want %d", effect, EffectMeteor)
	}
	if speed != 7 {
		t.Errorf("the speed is %d, want 7", speed)
	}
	if direction != 1 {
		t.Errorf("the direction is %d, want 1", direction)
	}
}

func TestSetLightingKeepsTheSwitchOffOfTheBacklight(t *testing.T) {
	// Changing the lighting must not change the automatic switch off of
	// the backlight.
	provider := newFakeProvider()
	// The action class reports the backlight with the timeout in the
	// sixth byte.
	provider.output("GetFunction", "uiOutput", uint64(backlightTimeoutSeconds)<<40)
	provider.output("GetGamingRgbKb", "gmOutput", 0x3200)
	backend := newFakeBackend(provider)

	if err := backend.SetLighting(RGBConfig{Brightness: 80}); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetFunction")
	if !ok {
		t.Fatal("the backlight was not written")
	}
	timeout := (call.Input >> 40) & 0xFF
	if timeout != backlightTimeoutSeconds {
		t.Errorf("the timeout is %d, want %d", timeout, backlightTimeoutSeconds)
	}
}

func TestSetLightingStopsAtTheFirstError(t *testing.T) {
	provider := newFakeProvider()
	provider.fail(errors.New("the provider refuses"))
	backend := newFakeBackend(provider)

	if err := backend.SetLighting(RGBConfig{}); err == nil {
		t.Error("the call was accepted although the provider refuses")
	}
}

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

func TestLightingBrightnessReadsTheSecondByte(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingRgbKb", "gmOutput", 0x3200)
	backend := newFakeBackend(provider)

	brightness, err := backend.LightingBrightness()
	if err != nil {
		t.Fatal(err)
	}
	if brightness != 50 {
		t.Errorf("the brightness is %d, want 50", brightness)
	}
}

func TestLightingBrightnessOfAnUninitialisedKeyboard(t *testing.T) {
	// The provider clamps the level, a larger value is an uninitialised
	// state rather than a brighter keyboard.
	provider := newFakeProvider()
	provider.output("GetGamingRgbKb", "gmOutput", 0xFFFF00)
	backend := newFakeBackend(provider)

	brightness, err := backend.LightingBrightness()
	if err != nil {
		t.Fatal(err)
	}
	if brightness != 100 {
		t.Errorf("the brightness is %d, want 100", brightness)
	}
}

func TestLightingReportsTheColoursAsUnreadable(t *testing.T) {
	// The firmware reports the brightness only, the colours and the
	// effect are not reported back.
	provider := newFakeProvider()
	provider.output("GetGamingRgbKb", "gmOutput", 0x3200)
	backend := newFakeBackend(provider)

	config, err := backend.Lighting()
	if err != nil {
		t.Fatal(err)
	}
	if config.Brightness != 50 {
		t.Errorf("the brightness is %d, want 50", config.Brightness)
	}
	if config.ColourReadable {
		t.Error("the firmware reports the colours")
	}
	if config.EffectReadable {
		t.Error("the firmware reports the effect")
	}
}

func TestSetBacklightWritesTheLevelInBothClasses(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if err := backend.SetBacklight(42, true); err != nil {
		t.Fatal(err)
	}

	// The generic action class takes the level in the fifth byte.
	call, ok := provider.callOf("SetFunction")
	if !ok {
		t.Fatal("the backlight was not written")
	}
	level := (call.Input >> 32) & 0xFF
	if level != 42 {
		t.Errorf("the level is %d, want 42", level)
	}

	// The gaming class takes the level as a buffer of sixteen bytes.
	buffer, ok := provider.callOf("SetGamingKBBacklight")
	if !ok {
		t.Fatal("the buffer of the backlight was not written")
	}
	if len(buffer.Bytes) != 16 {
		t.Fatalf("the buffer is %d bytes long, want 16", len(buffer.Bytes))
	}
	if buffer.Bytes[2] != 42 {
		t.Errorf("the level of the buffer is %d, want 42", buffer.Bytes[2])
	}
}

func TestSetBacklightClampsTheLevel(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if err := backend.SetBacklight(199, false); err != nil {
		t.Fatal(err)
	}

	call, _ := provider.callOf("SetFunction")
	level := (call.Input >> 32) & 0xFF
	if level != 100 {
		t.Errorf("the level is %d, want 100", level)
	}
}

func TestSetBacklightWithoutTheTimeout(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if err := backend.SetBacklight(50, false); err != nil {
		t.Fatal(err)
	}

	call, _ := provider.callOf("SetFunction")
	timeout := (call.Input >> 40) & 0xFF
	if timeout != 0 {
		t.Errorf("the timeout is %d, want 0", timeout)
	}
}

func TestBacklightReadsTheTimeout(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingRgbKb", "gmOutput", 0x3200)
	// The action class carries the timeout in the sixth byte.
	provider.output("GetFunction", "uiOutput", uint64(backlightTimeoutSeconds)<<40)
	backend := newFakeBackend(provider)

	state, err := backend.Backlight()
	if err != nil {
		t.Fatal(err)
	}
	if state.Brightness != 50 {
		t.Errorf("the brightness is %d, want 50", state.Brightness)
	}
	if !state.Timeout {
		t.Error("the backlight does not switch off on its own")
	}
}

func TestBacklightWithoutTheTimeout(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingRgbKb", "gmOutput", 0x3200)
	provider.output("GetFunction", "uiOutput", 0x00000000)
	backend := newFakeBackend(provider)

	state, err := backend.Backlight()
	if err != nil {
		t.Fatal(err)
	}
	if state.Timeout {
		t.Error("the backlight switches off on its own")
	}
}
