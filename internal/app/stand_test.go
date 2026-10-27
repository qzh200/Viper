package app

import (
	"testing"

	"viper/internal/hardware"
)

// 替身写完之后读回来必须是新值。
//
// 真实的机器是这样：写进去，再读回来，读到的是刚写的。替身如果写完不动
// 状态，读写一致性就没有被测过，写错了也发现不了。

func TestTheStandReadsBackWhatWasWritten(t *testing.T) {
	application, _ := newTestAppWith(t)

	// 背光。
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

	// 风扇。
	if err := application.SetFans(hardware.FanConfig{
		CPUMode:    hardware.FanCustom,
		CPUPercent: 60,
	}); err != nil {
		t.Fatal(err)
	}
	fans, err := application.Fans()
	if err != nil {
		t.Fatal(err)
	}
	if fans.CPUMode != hardware.FanCustom {
		t.Errorf("the fan was written in the mode %s but reads back as %s", hardware.FanCustom, fans.CPUMode)
	}
	if fans.CPUPercent != 60 {
		t.Errorf("the fan was written at 60 percent but reads back at %d", fans.CPUPercent)
	}

	// 灯光。
	lighting := hardware.RGBConfig{
		Effect:     hardware.EffectWave,
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
	if read.Effect != hardware.EffectWave {
		t.Errorf("the effect was written as %d but reads back as %d", hardware.EffectWave, read.Effect)
	}

	// 档位。
	if err := application.SetPowerMode(hardware.PowerTurbo); err != nil {
		t.Fatal(err)
	}
	mode, err := application.PowerMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != hardware.PowerTurbo {
		t.Errorf("the profile was written as %s but reads back as %s", hardware.PowerTurbo, mode)
	}
}
