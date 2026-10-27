package service

import (
	"testing"

	"viper/internal/config"
	"viper/internal/hardware"
)

// The official application sends the settings as a set of values. The
// numbers arrive as a float, because they come out of a JSON document.

func TestNumberOfTheKinds(t *testing.T) {
	cases := []struct {
		name   string
		values any
		want   int64
	}{
		{"a float", map[string]any{"mode": float64(4)}, 4},
		{"an integer", map[string]any{"mode": int(4)}, 4},
		{"a long integer", map[string]any{"mode": int64(4)}, 4},
		{"a text", map[string]any{"mode": "4"}, 4},
		{"a negative number", map[string]any{"mode": float64(-1)}, -1},
	}
	for _, c := range cases {
		got, err := number(c.values, "mode")
		if err != nil {
			t.Errorf("%s was refused: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s is %d, want %d", c.name, got, c.want)
		}
	}
}

func TestNumberOfSomethingElse(t *testing.T) {
	cases := []struct {
		name   string
		values any
	}{
		{"a set of values that is not one", "mode"},
		{"a missing value", map[string]any{"other": float64(1)}},
		{"a text that is not a number", map[string]any{"mode": "four"}},
		{"a switch", map[string]any{"mode": true}},
	}
	for _, c := range cases {
		if _, err := number(c.values, "mode"); err == nil {
			t.Errorf("%s was taken for a number", c.name)
		}
	}
}

func TestParseColorOfTheColours(t *testing.T) {
	cases := []struct {
		text string
		want hardware.RGBColor
	}{
		{"#C70087", hardware.RGBColor{R: 0xC7, G: 0, B: 0x87}},
		{"C70087", hardware.RGBColor{R: 0xC7, G: 0, B: 0x87}},
		{"#c70087", hardware.RGBColor{R: 0xC7, G: 0, B: 0x87}},
		{"  #00aec7  ", hardware.RGBColor{R: 0, G: 0xAE, B: 0xC7}},
	}
	for _, c := range cases {
		got, ok := parseColor(c.text)
		if !ok {
			t.Errorf("the colour %q was refused", c.text)
			continue
		}
		if got != c.want {
			t.Errorf("the colour %q is %+v, want %+v", c.text, got, c.want)
		}
	}
}

func TestParseColorOfTheOtherTexts(t *testing.T) {
	cases := []string{
		"",
		"#",
		"#C7008",
		"#C700877",
		"#GGGGGG",
		"nonsense",
		"#-10000",
	}
	for _, text := range cases {
		if _, ok := parseColor(text); ok {
			t.Errorf("the text %q was taken for a colour", text)
		}
	}
}

func TestLightingEffectValueOfTheNames(t *testing.T) {
	cases := map[string]hardware.LightingEffect{
		"DIRECT":    hardware.EffectStatic,
		"STATIC":    hardware.EffectStatic,
		"BREATHING": hardware.EffectBreathing,
		"NEON":      hardware.EffectNeon,
		"WAVE":      hardware.EffectWave,
		"SHIFTING":  hardware.EffectSwitch,
		"ZOOM":      hardware.EffectScale,
		"METEOR":    hardware.EffectMeteor,
		"TWINKLING": hardware.EffectFlash,
	}
	for name, want := range cases {
		got, ok := lightingEffectValue(name)
		if !ok {
			t.Errorf("the pattern %q was refused", name)
			continue
		}
		if got != want {
			t.Errorf("the pattern %q is %d, want %d", name, got, want)
		}
	}
}

func TestLightingEffectValueIsNotCaseSensitive(t *testing.T) {
	// The official application writes the names in capitals, but a client
	// that writes them in small letters is not refused.
	got, ok := lightingEffectValue("wave")
	if !ok {
		t.Fatal("the pattern in small letters was refused")
	}
	if got != hardware.EffectWave {
		t.Errorf("the pattern in small letters is %d, want %d", got, hardware.EffectWave)
	}
}

func TestLightingEffectValueOfAnUnknownName(t *testing.T) {
	got, ok := lightingEffectValue("SPARKLE")
	if ok {
		t.Error("an unknown pattern was accepted")
	}
	if got != hardware.EffectStatic {
		t.Errorf("an unknown pattern is %d, want the static pattern", got)
	}
}

func TestStoredLightingKeepsTheValues(t *testing.T) {
	value := hardware.RGBConfig{
		Effect:     hardware.EffectMeteor,
		Speed:      7,
		Direction:  1,
		Brightness: 80,
		PerZone:    true,
	}
	value.Zones[0] = hardware.RGBColor{R: 199, G: 0, B: 135}

	stored := storedLighting(value)
	if stored.Effect != uint64(hardware.EffectMeteor) {
		t.Errorf("the stored effect is %d, want %d", stored.Effect, hardware.EffectMeteor)
	}
	if stored.Speed != 7 || stored.Direction != 1 || stored.Brightness != 80 {
		t.Errorf("the stored lighting is %+v, want the values of the request", stored)
	}
	if !stored.PerZone {
		t.Error("the per zone flag was lost")
	}
	if stored.Zones[0] != (config.LightingColor{R: 199, G: 0, B: 135}) {
		t.Errorf("the first zone is %+v, want the magenta", stored.Zones[0])
	}
}

func TestApplyOfAnUnknownSetting(t *testing.T) {
	provider := NewProvider(nil, nil)
	if err := provider.Apply("NOTHING", map[string]any{"status": float64(1)}); err == nil {
		t.Error("an unknown setting was written")
	}
}

func TestApplyOfTheOperatingMode(t *testing.T) {
	// Without the gaming provider the setting cannot be written, but the
	// profile has to be understood first, so that a wrong profile is
	// reported as such.
	provider := NewProvider(nil, nil)

	known := []float64{0, 1, 4, 5}
	for _, value := range known {
		err := provider.Apply("OPERATING_MODE", map[string]any{"mode": value})
		if err == nil {
			t.Errorf("the profile %v was written without the gaming provider", value)
		}
	}

	// A profile that the firmware does not know is refused before the
	// provider is asked.
	err := provider.Apply("OPERATING_MODE", map[string]any{"mode": float64(2)})
	if err == nil {
		t.Error("an unknown profile was accepted")
	}
}

func TestApplyOfTheLightingWithoutTheValues(t *testing.T) {
	provider := NewProvider(nil, nil)
	if err := provider.Apply("LIGHTING", "wave"); err == nil {
		t.Error("the lighting was written without a set of values")
	}
}

func TestApplyOfTheLightingWithoutTheGamingProvider(t *testing.T) {
	provider := NewProvider(nil, nil)
	err := provider.Apply("LIGHTING", map[string]any{
		"effect": "WAVE",
		"speed":  float64(5),
	})
	if err == nil {
		t.Error("the lighting was written without the gaming provider")
	}
}

func TestApplyOfTheStickyKeysWithoutTheValues(t *testing.T) {
	provider := NewProvider(nil, nil)
	if err := provider.Apply("STICKY_KEY", map[string]any{}); err == nil {
		t.Error("the sticky keys were written without a status")
	}
}
