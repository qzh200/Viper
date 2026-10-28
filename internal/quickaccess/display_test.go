package quickaccess

import "testing"

func TestParseDisplayMode(t *testing.T) {
	cases := map[string]DisplayMode{
		"hybrid":    DisplayModeHybrid,
		"optimus":   DisplayModeHybrid,
		"mshybrid":  DisplayModeHybrid,
		"discrete":  DisplayModeDiscrete,
		"dgpu":      DisplayModeDiscrete,
		"nvidia":    DisplayModeDiscrete,
		"only":      DisplayModeDiscrete,
		"auto":      DisplayModeAuto,
		"automatic": DisplayModeAuto,
		"dds":       DisplayModeAuto,
	}

	for text, want := range cases {
		got, ok := ParseDisplayMode(text)
		if !ok {
			t.Errorf("%q 该认得", text)
			continue
		}
		if got != want {
			t.Errorf("%q 是 %d，不是 %d", text, got, want)
		}
	}

	for _, text := range []string{"", "bogus", "独显"} {
		if _, ok := ParseDisplayMode(text); ok {
			t.Errorf("%q 不该认得", text)
		}
	}
}
