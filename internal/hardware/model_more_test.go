package hardware

import (
	"testing"

	"viper/internal/wmi"
)

// The Acer provider returns most readings as a hexadecimal string, but
// some of them arrive as a number, so both have to be accepted.

func TestDecodeUintOfTheNumbers(t *testing.T) {
	cases := []struct {
		value wmi.Value
		want  uint64
	}{
		{wmi.Uint64(42), 42},
		{wmi.Uint32(42), 42},
		{wmi.Int32(-1), 0xFFFFFFFF},
	}
	for _, c := range cases {
		got, err := decodeUint(c.value)
		if err != nil {
			t.Errorf("%v was refused: %v", c.value, err)
			continue
		}
		if got != c.want {
			t.Errorf("%v is %d, want %d", c.value, got, c.want)
		}
	}
}

func TestDecodeUintOfTheTexts(t *testing.T) {
	cases := []struct {
		text string
		want uint64
	}{
		{"42", 42},
		{"0x2A", 42},
		{"0X2A", 42},
		{"  42  ", 42},
		{"  0x2A  ", 42},
		{"0xFFFFFFFF", 0xFFFFFFFF},
		{"", 0},
		{"   ", 0},
	}
	for _, c := range cases {
		got, err := decodeUint(wmi.String(c.text))
		if err != nil {
			t.Errorf("the text %q was refused: %v", c.text, err)
			continue
		}
		if got != c.want {
			t.Errorf("the text %q is %d, want %d", c.text, got, c.want)
		}
	}
}

func TestDecodeUintOfABrokenText(t *testing.T) {
	cases := []string{"nonsense", "0x", "12.5", "-1"}
	for _, text := range cases {
		if _, err := decodeUint(wmi.String(text)); err == nil {
			t.Errorf("the text %q was taken for a number", text)
		}
	}
}

func TestFanModeNames(t *testing.T) {
	cases := map[FanMode]string{
		FanAuto:   "auto",
		FanMax:    "max",
		FanCustom: "custom",
	}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("the mode %d renders as %q, want %q", mode, got, want)
		}
	}
	if got := FanMode(99).String(); got != "unknown" {
		t.Errorf("an unknown mode renders as %q, want unknown", got)
	}
}

func TestPowerModeNames(t *testing.T) {
	cases := map[PowerMode]string{
		PowerQuiet:       "quiet",
		PowerBalanced:    "balanced",
		PowerPerformance: "performance",
		PowerTurbo:       "turbo",
		PowerUnknown:     "unknown",
	}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("the mode %d renders as %q, want %q", mode, got, want)
		}
	}
	if got := PowerMode(99).String(); got != "unknown" {
		t.Errorf("an unknown mode renders as %q, want unknown", got)
	}
}

func TestGPUModeNames(t *testing.T) {
	cases := map[GPUMode]string{
		GPUModeHybrid:   "hybrid",
		GPUModeDiscrete: "discrete",
		GPUModeAuto:     "auto",
	}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("the mode %d renders as %q, want %q", mode, got, want)
		}
	}
	if got := GPUMode(99).String(); got != "unknown" {
		t.Errorf("an unknown mode renders as %q, want unknown", got)
	}
}

func TestLightingEffectNameOfAnUnknownEffect(t *testing.T) {
	if got := LightingEffect(99).String(); got != "unknown" {
		t.Errorf("an unknown effect renders as %q, want unknown", got)
	}
}

func TestParsePowerModeOfTheOtherNames(t *testing.T) {
	// The command line takes a few spellings of every profile.
	cases := map[string]PowerMode{
		"default":  PowerBalanced,
		"perf":     PowerPerformance,
		"extreme":  PowerTurbo,
		"balanced": PowerBalanced,
	}
	for name, want := range cases {
		got, ok := ParsePowerMode(name)
		if !ok {
			t.Errorf("the name %q was refused", name)
			continue
		}
		if got != want {
			t.Errorf("the name %q is %s, want %s", name, got, want)
		}
	}
}

func TestParsePowerModeOfANameThatIsNotThere(t *testing.T) {
	got, ok := ParsePowerMode("loud")
	if ok {
		t.Error("a name that is not there was accepted")
	}
	if got != PowerBalanced {
		t.Errorf("the fallback is %s, want the balanced profile", got)
	}
}

func TestParseLightingEffectOfTheOtherNames(t *testing.T) {
	cases := map[string]LightingEffect{
		"breathe":      EffectBreathing,
		"alternate":    EffectSwitch,
		"zoom":         EffectScale,
		"meteorshower": EffectMeteor,
		"strobe":       EffectFlash,
	}
	for name, want := range cases {
		got, ok := ParseLightingEffect(name)
		if !ok {
			t.Errorf("the name %q was refused", name)
			continue
		}
		if got != want {
			t.Errorf("the name %q is %d, want %d", name, got, want)
		}
	}
}

func TestParseLightingEffectOfANameThatIsNotThere(t *testing.T) {
	got, ok := ParseLightingEffect("sparkle")
	if ok {
		t.Error("a name that is not there was accepted")
	}
	if got != EffectStatic {
		t.Errorf("the fallback is %d, want the static pattern", got)
	}
}

func TestParseGPUModeOfTheOtherNames(t *testing.T) {
	cases := map[string]GPUMode{
		"optimus":   GPUModeHybrid,
		"mshybrid":  GPUModeHybrid,
		"dgpu":      GPUModeDiscrete,
		"nvidia":    GPUModeDiscrete,
		"only":      GPUModeDiscrete,
		"automatic": GPUModeAuto,
		"dds":       GPUModeAuto,
	}
	for name, want := range cases {
		got, ok := ParseGPUMode(name)
		if !ok {
			t.Errorf("the name %q was refused", name)
			continue
		}
		if got != want {
			t.Errorf("the name %q is %s, want %s", name, got, want)
		}
	}
}

func TestParseGPUModeOfANameThatIsNotThere(t *testing.T) {
	got, ok := ParseGPUMode("sideways")
	if ok {
		t.Error("a name that is not there was accepted")
	}
	if got != GPUModeHybrid {
		t.Errorf("the fallback is %s, want the hybrid mode", got)
	}
}

func TestAvailablePowerModesOfAPredator(t *testing.T) {
	// The turbo profile is only offered on a Predator.
	modes := AvailablePowerModes(true)
	want := []PowerMode{PowerQuiet, PowerBalanced, PowerPerformance, PowerTurbo, PowerEco}
	if len(modes) != len(want) {
		t.Fatalf("the profiles are %v, want %v", modes, want)
	}
	for i := range want {
		if modes[i] != want[i] {
			t.Errorf("the profile %d is %s, want %s", i, modes[i], want[i])
		}
	}
}

func TestAvailablePowerModesOfAnotherMachine(t *testing.T) {
	modes := AvailablePowerModes(false)
	for _, mode := range modes {
		if mode == PowerTurbo {
			t.Errorf("the profiles are %v, want no turbo profile", modes)
		}
	}
	if len(modes) != 4 {
		t.Errorf("the profiles are %v, want four of them", modes)
	}
}

func TestDecodeFanModeOfAValueThatIsNotThere(t *testing.T) {
	// The embedded controller answers with the automatic mode when it does
	// not know the behaviour.
	if got := decodeFanMode(0); got != FanAuto {
		t.Errorf("the value 0 decodes to %s, want the automatic mode", got)
	}
	if got := decodeFanMode(99); got != FanAuto {
		t.Errorf("the value 99 decodes to %s, want the automatic mode", got)
	}
}

func TestDecodePowerModeOfAValueThatIsNotThere(t *testing.T) {
	// The provider keeps a sixth value, it is the profile that saves the
	// power. Every other value is unknown.
	if got := decodePowerMode(6); got != PowerEco {
		t.Errorf("the value 6 decodes to %s, want the eco profile", got)
	}
	if got := decodePowerMode(99); got != PowerUnknown {
		t.Errorf("the value 99 decodes to %s, want the unknown profile", got)
	}
}

func TestIsPredatorModelOfTheOtherMachines(t *testing.T) {
	// Only the Predator carries the turbo profile, so the family of the
	// machine decides what the program offers.
	cases := []struct {
		model  string
		family string
		want   bool
	}{
		{"Predator PHN16-71", "Predator", true},
		{"PHN16-71", "predator", true},
		{"PREDATOR", "", true},
		{"Nitro AN515-57", "Nitro", false},
		{"Swift Go 14", "Swift", false},
		{"", "", false},
	}
	for _, c := range cases {
		if got := IsPredatorModel(c.model, c.family); got != c.want {
			t.Errorf("the machine %q of the family %q is a Predator: %v, want %v",
				c.model, c.family, got, c.want)
		}
	}
}
