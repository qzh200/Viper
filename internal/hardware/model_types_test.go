package hardware

import (
	"testing"
)

// 这一层现在是机器的词汇表：性能档位、灯效、显卡模式、风扇与传感器的类型。
// 这些用例守着它们的编号与名字——原厂服务与固件用的是同一套编号，编错一个
// 数字，机器就会理解成另一个意思。

func TestTheNamesOfTheFanModes(t *testing.T) {
	cases := map[FanMode]string{
		FanAuto:     "auto",
		FanMax:      "max",
		FanCustom:   "custom",
		FanMode(99): "unknown",
	}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("风扇模式 %d 的名字是 %q，不是 %q", mode, got, want)
		}
	}
}

func TestTheNamesOfThePerformanceProfiles(t *testing.T) {
	cases := map[PowerMode]string{
		PowerQuiet:       "quiet",
		PowerBalanced:    "balanced",
		PowerPerformance: "performance",
		PowerTurbo:       "turbo",
		PowerUnknown:     "unknown",
		PowerEco:         "eco",
	}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("档位 %d 的名字是 %q，不是 %q", mode, got, want)
		}
	}
}

func TestTheNumbersOfThePerformanceProfiles(t *testing.T) {
	// 原厂服务与固件用的是同一套编号：0 静音、1 均衡、4 性能、5 加速、6 节能。
	cases := map[PowerMode]uint64{
		PowerQuiet:       0,
		PowerBalanced:    1,
		PowerPerformance: 4,
		PowerTurbo:       5,
		PowerEco:         6,
	}
	for mode, want := range cases {
		if got := PowerModeCode(mode); got != want {
			t.Errorf("%v 的编号是 %d，不是 %d", mode, got, want)
		}
		if got := DecodePowerMode(want); got != mode {
			t.Errorf("编号 %d 解出来是 %v，不是 %v", want, got, mode)
		}
	}

	// 不认识的编号只能解成"不知道"。
	if got := DecodePowerMode(2); got != PowerUnknown {
		t.Errorf("编号 2 解出来是 %v，不是不知道", got)
	}
	if got := DecodePowerMode(99); got != PowerUnknown {
		t.Errorf("编号 99 解出来是 %v，不是不知道", got)
	}

	// 不认识的档位按均衡写。
	if got := PowerModeCode(PowerUnknown); got != 1 {
		t.Errorf("不知道的档位写成 %d，不是 1", got)
	}
}

func TestTheProfilesOfAMachine(t *testing.T) {
	quiet := AvailablePowerModes(false)
	found := false
	for _, mode := range quiet {
		if mode == PowerQuiet {
			found = true
		}
	}
	if !found {
		t.Errorf("每一台机器都有静音档，%v 里没有", quiet)
	}

	predator := AvailablePowerModes(true)
	found = false
	for _, mode := range predator {
		if mode == PowerTurbo {
			found = true
		}
	}
	if !found {
		t.Errorf("Predator 有加速档，%v 里没有", predator)
	}
}

func TestTheNamesOfThePerformanceProfilesOnTheCommandLine(t *testing.T) {
	cases := map[string]PowerMode{
		"quiet":       PowerQuiet,
		"balanced":    PowerBalanced,
		"default":     PowerBalanced,
		"performance": PowerPerformance,
		"perf":        PowerPerformance,
		"turbo":       PowerTurbo,
		"extreme":     PowerTurbo,
		"eco":         PowerEco,
		"power_saver": PowerEco,
	}
	for name, want := range cases {
		got, ok := ParsePowerMode(name)
		if !ok {
			t.Errorf("档位 %q 解析不出来", name)
			continue
		}
		if got != want {
			t.Errorf("档位 %q 解析成 %v，不是 %v", name, got, want)
		}
	}

	if _, ok := ParsePowerMode("nonsense"); ok {
		t.Error("不存在的档位不该解析成功")
	}
}

func TestTheNamesOfTheLightingPatterns(t *testing.T) {
	cases := map[LightingEffect]string{
		EffectStatic:       "static",
		EffectBreathing:    "breathing",
		EffectNeon:         "neon",
		EffectWave:         "wave",
		EffectSwitch:       "switch",
		EffectScale:        "scale",
		EffectMeteor:       "meteor",
		EffectFlash:        "flash",
		LightingEffect(99): "unknown",
	}
	for effect, want := range cases {
		if got := effect.String(); got != want {
			t.Errorf("灯效 %d 的名字是 %q，不是 %q", effect, got, want)
		}
	}
}

func TestTheNamesOfTheLightingPatternsOnTheCommandLine(t *testing.T) {
	cases := map[string]LightingEffect{
		"static":       EffectStatic,
		"breathing":    EffectBreathing,
		"breathe":      EffectBreathing,
		"neon":         EffectNeon,
		"wave":         EffectWave,
		"switch":       EffectSwitch,
		"alternate":    EffectSwitch,
		"scale":        EffectScale,
		"zoom":         EffectScale,
		"meteor":       EffectMeteor,
		"meteorshower": EffectMeteor,
		"flash":        EffectFlash,
		"strobe":       EffectFlash,
	}
	for name, want := range cases {
		got, ok := ParseLightingEffect(name)
		if !ok {
			t.Errorf("灯效 %q 解析不出来", name)
			continue
		}
		if got != want {
			t.Errorf("灯效 %q 解析成 %v，不是 %v", name, got, want)
		}
	}

	if _, ok := ParseLightingEffect("nonsense"); ok {
		t.Error("不存在的灯效不该解析成功")
	}
}

func TestTheNamesOfTheGraphicsModes(t *testing.T) {
	cases := map[GPUMode]string{
		GPUModeHybrid:   "hybrid",
		GPUModeDiscrete: "discrete",
		GPUModeAuto:     "auto",
		GPUMode(99):     "unknown",
	}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("显卡模式 %d 的名字是 %q，不是 %q", mode, got, want)
		}
	}
}

func TestTheNamesOfTheGraphicsModesOnTheCommandLine(t *testing.T) {
	cases := map[string]GPUMode{
		"hybrid":    GPUModeHybrid,
		"optimus":   GPUModeHybrid,
		"mshybrid":  GPUModeHybrid,
		"discrete":  GPUModeDiscrete,
		"dgpu":      GPUModeDiscrete,
		"nvidia":    GPUModeDiscrete,
		"only":      GPUModeDiscrete,
		"auto":      GPUModeAuto,
		"automatic": GPUModeAuto,
		"dds":       GPUModeAuto,
	}
	for name, want := range cases {
		got, ok := ParseGPUMode(name)
		if !ok {
			t.Errorf("显卡模式 %q 解析不出来", name)
			continue
		}
		if got != want {
			t.Errorf("显卡模式 %q 解析成 %v，不是 %v", name, got, want)
		}
	}

	if _, ok := ParseGPUMode("nonsense"); ok {
		t.Error("不存在的显卡模式不该解析成功")
	}
}

func TestTheNumbersAreLimited(t *testing.T) {
	if got := clampUint(5, 1, 9); got != 5 {
		t.Errorf("范围之内的数是 %d，不是 5", got)
	}
	if got := clampUint(0, 1, 9); got != 1 {
		t.Errorf("太小的数是 %d，不是 1", got)
	}
	if got := clampUint(20, 1, 9); got != 9 {
		t.Errorf("太大的数是 %d，不是 9", got)
	}
}
