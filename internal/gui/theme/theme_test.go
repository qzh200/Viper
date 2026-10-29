package theme

import "testing"

func TestParseKnowsTheThreeModes(t *testing.T) {
	cases := map[string]Mode{
		"auto":  ModeAuto,
		"light": ModeLight,
		"dark":  ModeDark,
	}
	for text, want := range cases {
		got, ok := Parse(text)
		if !ok || got != want {
			t.Errorf("Parse(%q) = %q, %v, want %q, true", text, got, ok, want)
		}
	}
}

func TestParseIsForgivingAboutCaseAndSpace(t *testing.T) {
	got, ok := Parse("  LIGHT ")
	if !ok || got != ModeLight {
		t.Errorf("Parse(\"  LIGHT \") = %q, %v, want %q, true", got, ok, ModeLight)
	}
}

func TestParseRefusesWhatItDoesNotKnow(t *testing.T) {
	for _, text := range []string{"", "blue", "auto2", "深色"} {
		if got, ok := Parse(text); ok {
			t.Errorf("Parse(%q) = %q, true, want the automatic mode and false", text, got)
		}
	}
}

func TestValid(t *testing.T) {
	for _, mode := range Modes() {
		if !mode.Valid() {
			t.Errorf("the mode %q is not valid", mode)
		}
	}
	if Mode("sepia").Valid() {
		t.Error("an unknown mode is valid")
	}
}

func TestLabel(t *testing.T) {
	cases := map[Mode]string{
		ModeAuto:  "自动",
		ModeLight: "亮色",
		ModeDark:  "暗色",
	}
	for mode, want := range cases {
		if got := mode.Label(); got != want {
			t.Errorf("the label of %q is %q, want %q", mode, got, want)
		}
	}
}

func TestLabelsFollowTheModes(t *testing.T) {
	modes := Modes()
	labels := Labels()
	if len(labels) != len(modes) {
		t.Fatalf("there are %d labels and %d modes", len(labels), len(modes))
	}
	for i, mode := range modes {
		if labels[i] != mode.Label() {
			t.Errorf("the label at %d is %q, want %q", i, labels[i], mode.Label())
		}
	}
}

func TestNextCyclesThroughTheModes(t *testing.T) {
	seen := map[Mode]bool{}
	mode := ModeAuto
	for i := 0; i < len(Modes()); i++ {
		seen[mode] = true
		mode = mode.Next()
	}
	if len(seen) != len(Modes()) {
		t.Errorf("the cycle went through %d modes, want %d", len(seen), len(Modes()))
	}
	if ModeAuto.Next() != ModeLight {
		t.Errorf("the automatic mode is followed by %q, want the light one", ModeAuto.Next())
	}
}

func TestNextOfAnUnknownModeIsTheAutomaticOne(t *testing.T) {
	if got := Mode("sepia").Next(); got != ModeAuto {
		t.Errorf("the next mode of an unknown one is %q, want the automatic one", got)
	}
}

func TestResolveFollowsTheSystemOnlyInTheAutomaticMode(t *testing.T) {
	cases := []struct {
		mode Mode
		dark bool
		want Appearance
	}{
		{ModeAuto, true, AppearanceDark},
		{ModeAuto, false, AppearanceLight},
		{ModeLight, true, AppearanceLight},
		{ModeLight, false, AppearanceLight},
		{ModeDark, true, AppearanceDark},
		{ModeDark, false, AppearanceDark},
	}
	for _, item := range cases {
		got := Resolve(item.mode, item.dark)
		if got != item.want {
			t.Errorf("Resolve(%q, %v) = %q, want %q", item.mode, item.dark, got, item.want)
		}
	}
}

func TestResolveOfAnUnknownModeFollowsTheSystem(t *testing.T) {
	if got := Resolve(Mode("sepia"), true); got != AppearanceDark {
		t.Errorf("Resolve of an unknown mode = %q, want the dark appearance", got)
	}
}

func TestAppearanceDark(t *testing.T) {
	if !AppearanceDark.Dark() {
		t.Error("the dark appearance is not dark")
	}
	if AppearanceLight.Dark() {
		t.Error("the light appearance is dark")
	}
}

func TestIndexAndAtWalkTheModes(t *testing.T) {
	modes := Modes()
	for i, mode := range modes {
		if got := Index(mode); got != i {
			t.Errorf("Index(%q) = %d, want %d", mode, got, i)
		}
		if got := At(i); got != mode {
			t.Errorf("At(%d) = %q, want %q", i, got, mode)
		}
	}
	if got := Index(Mode("sepia")); got != 0 {
		t.Errorf("Index of an unknown mode = %d, want 0", got)
	}
	if got := At(-1); got != ModeAuto {
		t.Errorf("At(-1) = %q, want the automatic mode", got)
	}
	if got := At(len(modes)); got != ModeAuto {
		t.Errorf("At past the end = %q, want the automatic mode", got)
	}
}

func TestModesAreACopy(t *testing.T) {
	modes := Modes()
	modes[0] = ModeDark
	if Modes()[0] != ModeAuto {
		t.Error("changing the returned modes changed the package")
	}
}
