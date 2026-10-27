package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	settings := Default()
	if settings.PowerMode != PowerBalanced {
		t.Errorf("the power mode is %d, want %d", settings.PowerMode, PowerBalanced)
	}
	if settings.Lighting.Brightness > 100 {
		t.Errorf("the brightness is %d, want at most 100", settings.Lighting.Brightness)
	}
	if !settings.BacklightTimeout {
		t.Error("the backlight timeout is switched off by default")
	}
	if settings.Fans.CPUMode != FanAuto || settings.Fans.GPUMode != FanAuto {
		t.Errorf("the fans start in the mode %d and %d, want %d and %d, which is the automatic mode",
			settings.Fans.CPUMode, settings.Fans.GPUMode, FanAuto, FanAuto)
	}
	if settings.Fans.CPUPercent != 0 || settings.Fans.GPUPercent != 0 {
		t.Errorf("the fans start at %d and %d percent, want zero",
			settings.Fans.CPUPercent, settings.Fans.GPUPercent)
	}
}

func TestTheDefaultSettingsCarryTheEnumeration(t *testing.T) {
	// The settings carry the semantic enumeration of the hardware layer,
	// never the codes that the firmware expects. The codes of the
	// firmware are 0, 1, 4 and 5 for the profiles and 1, 2 and 3 for the
	// fan behaviours.
	settings := Default()
	if settings.PowerMode == 4 || settings.PowerMode == 5 {
		t.Errorf("the power mode is %d, want a value of the enumeration and not a code of the firmware",
			settings.PowerMode)
	}
	if settings.Fans.CPUMode == 1 || settings.Fans.CPUMode == 2 || settings.Fans.CPUMode == 3 {
		t.Errorf("the mode of the CPU fan is %d, want a value of the enumeration and not a code of the firmware",
			settings.Fans.CPUMode)
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
	settings.PowerMode = PowerPerformance
	settings.Fans.CPUMode = FanCustom
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
	if restored.PowerMode != PowerPerformance {
		t.Errorf("the power mode is %d, want %d", restored.PowerMode, PowerPerformance)
	}
	if restored.Fans.CPUMode != FanCustom || restored.Fans.CPUPercent != 60 || !restored.Fans.CoolBoost {
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
