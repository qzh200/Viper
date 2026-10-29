package app

import (
	"testing"

	"viper/internal/domain"
)

func TestTheStandReadsBackWhatWasWritten(t *testing.T) {
	application, _ := newTestAppWith(t)

	if err := application.SetBacklightTimeout(true); err != nil {
		t.Fatal(err)
	}
	state, err := application.BacklightTimeout()
	if err != nil {
		t.Fatal(err)
	}
	if !state {
		t.Error("the backlight timeout was written but it reads back as off")
	}

	if err := application.SetFans(domain.FanConfig{
		CPUMode:    domain.FanCustom,
		CPUPercent: 60,
	}); err != nil {
		t.Fatal(err)
	}
	fans, err := application.Fans()
	if err != nil {
		t.Fatal(err)
	}
	if fans.CPUMode != domain.FanCustom {
		t.Errorf("the fan was written in the mode %s but reads back as %s", domain.FanCustom, fans.CPUMode)
	}
	if fans.CPUPercent != 60 {
		t.Errorf("the fan was written at 60 percent but reads back at %d", fans.CPUPercent)
	}

	lighting := domain.RGBConfig{
		Effect:     domain.EffectWave,
		Speed:      5,
		Direction:  1,
		Brightness: 80,
		PerZone:    true,
	}
	if err := application.SetLighting(lighting); err != nil {
		t.Fatal(err)
	}
	read, err := application.Lighting()
	if err != nil {
		t.Fatal(err)
	}
	if read.Effect != domain.EffectWave {
		t.Errorf("the effect was written as %d but reads back as %d", domain.EffectWave, read.Effect)
	}

	if err := application.SetPowerMode(domain.PowerTurbo); err != nil {
		t.Fatal(err)
	}
	mode, err := application.PowerMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != domain.PowerTurbo {
		t.Errorf("the profile was written as %s but reads back as %s", domain.PowerTurbo, mode)
	}
}
