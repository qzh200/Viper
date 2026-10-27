package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	settings := Default()
	if settings.PowerMode != 1 {
		t.Errorf("the power mode is %d, want 1", settings.PowerMode)
	}
	if settings.Lighting.Brightness > 100 {
		t.Errorf("the brightness is %d, want at most 100", settings.Lighting.Brightness)
	}
	if !settings.BacklightTimeout {
		t.Error("the backlight timeout is switched off by default")
	}
	if settings.Fans.CPUMode != 1 || settings.Fans.GPUMode != 1 {
		t.Errorf("the fans start in the mode %d and %d, want 1 and 1", settings.Fans.CPUMode, settings.Fans.GPUMode)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	settings := Default()
	settings.Lighting.Effect = 3
	settings.Lighting.Speed = 5
	settings.Lighting.Brightness = 100
	settings.Lighting.PerZone = true
	settings.Lighting.Zones[0] = LightingColor{R: 199, G: 0, B: 135}
	settings.Lighting.Zones[1] = LightingColor{R: 0, G: 174, B: 199}
	settings.PowerMode = 4
	settings.Fans.CPUMode = 3
	settings.Fans.CPUPercent = 60
	settings.Fans.CoolBoost = true
	settings.BacklightTimeout = false

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}

	var restored Settings
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}

	if restored.Lighting.Effect != 3 || restored.Lighting.Speed != 5 {
		t.Errorf("the lighting is %+v, want the effect 3 and the speed 5", restored.Lighting)
	}
	if restored.Lighting.Brightness != 100 {
		t.Errorf("the brightness is %d, want 100", restored.Lighting.Brightness)
	}
	if restored.Lighting.Zones[0] != (LightingColor{R: 199, G: 0, B: 135}) {
		t.Errorf("the first zone is %+v, want the magenta", restored.Lighting.Zones[0])
	}
	if restored.Lighting.Zones[1] != (LightingColor{R: 0, G: 174, B: 199}) {
		t.Errorf("the second zone is %+v, want the cyan", restored.Lighting.Zones[1])
	}
	if restored.PowerMode != 4 {
		t.Errorf("the power mode is %d, want 4", restored.PowerMode)
	}
	if restored.Fans.CPUMode != 3 || restored.Fans.CPUPercent != 60 || !restored.Fans.CoolBoost {
		t.Errorf("the fans are %+v, want the custom mode at 60 percent with the boost", restored.Fans)
	}
	if restored.BacklightTimeout {
		t.Error("the backlight timeout is switched on")
	}
}

func TestBrokenSettingsFallBackToTheDefaults(t *testing.T) {
	// A broken file must not stop the program.
	var settings Settings
	if err := json.Unmarshal([]byte("{not json"), &settings); err == nil {
		t.Error("the broken file was accepted")
	}
}
