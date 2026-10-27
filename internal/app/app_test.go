package app

import (
	"testing"

	"viper/internal/hardware"
)

func TestToStoredLightingLimitsTheValues(t *testing.T) {
	// The firmware accepts a brightness from zero to one hundred, a speed
	// from one to nine and a direction that is a switch. The stored
	// settings must never claim more than the keyboard can do, otherwise
	// the program shows a brightness that the keyboard does not have.
	config := hardware.RGBConfig{
		Effect:     hardware.EffectWave,
		Speed:      99,
		Direction:  5,
		Brightness: 199,
	}

	stored := toStoredLighting(config)
	if stored.Brightness != 100 {
		t.Errorf("the stored brightness is %d, want 100", stored.Brightness)
	}
	if stored.Speed != 9 {
		t.Errorf("the stored speed is %d, want 9", stored.Speed)
	}
	if stored.Direction != 1 {
		t.Errorf("the stored direction is %d, want 1", stored.Direction)
	}
}

func TestToStoredLightingKeepsValidValues(t *testing.T) {
	config := hardware.RGBConfig{
		Effect:     hardware.EffectBreathing,
		Speed:      5,
		Direction:  0,
		Brightness: 60,
	}

	stored := toStoredLighting(config)
	if stored.Brightness != 60 {
		t.Errorf("the stored brightness is %d, want 60", stored.Brightness)
	}
	if stored.Speed != 5 {
		t.Errorf("the stored speed is %d, want 5", stored.Speed)
	}
	if stored.Direction != 0 {
		t.Errorf("the stored direction is %d, want 0", stored.Direction)
	}
	if stored.Effect != uint64(hardware.EffectBreathing) {
		t.Errorf("the stored effect is %d, want %d", stored.Effect, hardware.EffectBreathing)
	}
}

func TestToStoredLightingKeepsTheColours(t *testing.T) {
	config := hardware.RGBConfig{
		PerZone: true,
	}
	config.Zones[0] = hardware.RGBColor{R: 199, G: 0, B: 135}
	config.Zones[1] = hardware.RGBColor{R: 0, G: 174, B: 199}

	stored := toStoredLighting(config)
	if stored.Zones[0].R != 199 || stored.Zones[0].G != 0 || stored.Zones[0].B != 135 {
		t.Errorf("the first zone is %+v, want the magenta of the original application", stored.Zones[0])
	}
	if stored.Zones[1].R != 0 || stored.Zones[1].G != 174 || stored.Zones[1].B != 199 {
		t.Errorf("the second zone is %+v, want the cyan of the original application", stored.Zones[1])
	}
	if !stored.PerZone {
		t.Error("the per zone flag was lost")
	}
}

func TestClamp(t *testing.T) {
	cases := []struct {
		value, low, high, want uint64
	}{
		{0, 0, 100, 0},
		{50, 0, 100, 50},
		{100, 0, 100, 100},
		{199, 0, 100, 100},
		{0, 1, 9, 1},
		{9, 1, 9, 9},
		{12, 1, 9, 9},
	}
	for _, c := range cases {
		if got := clamp(c.value, c.low, c.high); got != c.want {
			t.Errorf("clamp(%d, %d, %d) is %d, want %d", c.value, c.low, c.high, got, c.want)
		}
	}
}
