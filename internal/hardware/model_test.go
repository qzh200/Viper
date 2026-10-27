package hardware

import "testing"

// The tables in this file are the encodings that were measured on a Predator
// Helios Neo 16. They are the reason the code looks the way it does, so a
// change to one of them has to be deliberate.

func TestPowerModeProviderCode(t *testing.T) {
	// The provider keeps five values, two and three are silently clamped
	// back to the balanced profile. The MOF suggests a plain enumeration
	// and is wrong.
	cases := map[PowerMode]uint64{
		PowerQuiet:       0,
		PowerBalanced:    1,
		PowerPerformance: 4,
		PowerTurbo:       5,
	}
	for mode, want := range cases {
		if got := mode.providerCode(); got != want {
			t.Errorf("%s encodes to %d, want %d", mode, got, want)
		}
	}
}

func TestDecodePowerMode(t *testing.T) {
	cases := map[uint64]PowerMode{
		0: PowerQuiet,
		1: PowerBalanced,
		4: PowerPerformance,
		5: PowerTurbo,
		6: PowerEco,
	}
	for value, want := range cases {
		if got := decodePowerMode(value); got != want {
			t.Errorf("value %d decodes to %s, want %s", value, got, want)
		}
	}
}

func TestParsePowerMode(t *testing.T) {
	cases := map[string]PowerMode{
		"quiet":       PowerQuiet,
		"balanced":    PowerBalanced,
		"default":     PowerBalanced,
		"performance": PowerPerformance,
		"perf":        PowerPerformance,
		"turbo":       PowerTurbo,
		"extreme":     PowerTurbo,
	}
	for name, want := range cases {
		got, ok := ParsePowerMode(name)
		if !ok {
			t.Errorf("%q was not accepted", name)
			continue
		}
		if got != want {
			t.Errorf("%q decodes to %s, want %s", name, got, want)
		}
	}
	if _, ok := ParsePowerMode("nonsense"); ok {
		t.Error("nonsense was accepted")
	}
}

func TestDecodeFanMode(t *testing.T) {
	cases := map[uint64]FanMode{
		1: FanAuto,
		2: FanMax,
		3: FanCustom,
	}
	for value, want := range cases {
		if got := decodeFanMode(value); got != want {
			t.Errorf("value %d decodes to %s, want %s", value, got, want)
		}
	}
}

func TestLightingEffectNames(t *testing.T) {
	// The numbering comes from the profile file of the original software,
	// where every pattern carries a value field.
	cases := map[LightingEffect]string{
		EffectStatic:    "static",
		EffectBreathing: "breathing",
		EffectNeon:      "neon",
		EffectWave:      "wave",
		EffectSwitch:    "switch",
		EffectScale:     "scale",
		EffectMeteor:    "meteor",
		EffectFlash:     "flash",
	}
	for effect, want := range cases {
		if got := effect.String(); got != want {
			t.Errorf("effect %d renders as %q, want %q", effect, got, want)
		}
	}
}

func TestParseLightingEffect(t *testing.T) {
	cases := map[string]LightingEffect{
		"static":    EffectStatic,
		"breathing": EffectBreathing,
		"breathe":   EffectBreathing,
		"neon":      EffectNeon,
		"wave":      EffectWave,
		"switch":    EffectSwitch,
		"alternate": EffectSwitch,
		"scale":     EffectScale,
		"zoom":      EffectScale,
		"meteor":    EffectMeteor,
		"flash":     EffectFlash,
		"strobe":    EffectFlash,
	}
	for name, want := range cases {
		got, ok := ParseLightingEffect(name)
		if !ok {
			t.Errorf("%q was not accepted", name)
			continue
		}
		if got != want {
			t.Errorf("%q decodes to %s, want %s", name, got, want)
		}
	}
}

func TestGPUMode(t *testing.T) {
	// The numbering is the one of the display mode of the original service.
	cases := map[GPUMode]uint64{
		GPUModeHybrid:   1,
		GPUModeDiscrete: 2,
		GPUModeAuto:     3,
	}
	for mode, want := range cases {
		if got := uint64(mode); got != want {
			t.Errorf("%s encodes to %d, want %d", mode, got, want)
		}
	}
}

func TestParseGPUMode(t *testing.T) {
	cases := map[string]GPUMode{
		"hybrid":   GPUModeHybrid,
		"optimus":  GPUModeHybrid,
		"discrete": GPUModeDiscrete,
		"nvidia":   GPUModeDiscrete,
		"auto":     GPUModeAuto,
		"dds":      GPUModeAuto,
	}
	for name, want := range cases {
		got, ok := ParseGPUMode(name)
		if !ok {
			t.Errorf("%q was not accepted", name)
			continue
		}
		if got != want {
			t.Errorf("%q decodes to %s, want %s", name, got, want)
		}
	}
}

func TestMiscSettingIdentifiers(t *testing.T) {
	// The identifiers were found by writing every value of the table and
	// reading the original service back.
	cases := map[MiscSetting]uint32{
		MiscGPUMode:           0x02,
		MiscBootSound:         0x06,
		MiscGPUModeCapability: 0x09,
		MiscCPUOverclock:      0x0A,
		MiscOperationMode:     0x0B,
	}
	for setting, want := range cases {
		if got := uint32(setting); got != want {
			t.Errorf("setting 0x%02X was encoded as 0x%02X", want, got)
		}
	}
}

func TestClampPercent(t *testing.T) {
	cases := map[int]int{
		-1:  0,
		0:   0,
		50:  50,
		100: 100,
		199: 100,
	}
	for input, want := range cases {
		if got := clampPercent(input); got != want {
			t.Errorf("%d was clamped to %d, want %d", input, got, want)
		}
	}
}

func TestIsPredatorModel(t *testing.T) {
	if !IsPredatorModel("Predator PHN16-71", "") {
		t.Error("the Predator was not recognised")
	}
	if !IsPredatorModel("", "Predator") {
		t.Error("the Predator family was not recognised")
	}
	if IsPredatorModel("Nitro AN515-57", "Nitro") {
		t.Error("a Nitro was taken for a Predator")
	}
}
