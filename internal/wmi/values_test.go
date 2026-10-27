package wmi

import "testing"

func TestValueUInt64(t *testing.T) {
	cases := []struct {
		value Value
		want  uint64
	}{
		{Uint64(0x6400), 0x6400},
		{Uint32(0x2A), 42},
		{Int32(-1), 0xFFFFFFFF},
		{String("42"), 42},
		{String("0x100"), 256},
		{String("not a number"), 0},
	}
	for _, c := range cases {
		if got := c.value.UInt64(); got != c.want {
			t.Errorf("%v is %d, want %d", c.value, got, c.want)
		}
	}
}

func TestValueString(t *testing.T) {
	if got := Uint64(42).String(); got != "42" {
		t.Errorf("the number renders as %q, want 42", got)
	}
	if got := String("hello").String(); got != "hello" {
		t.Errorf("the text renders as %q, want hello", got)
	}
	if got := Bytes([]byte{1, 2, 3}).String(); got != "3 bytes" {
		t.Errorf("the array renders as %q, want 3 bytes", got)
	}
}

func TestValueCandidates(t *testing.T) {
	// The provider declares most parameters as unsigned integers but only
	// accepts a signed integer or a string, so the candidates are tried in
	// that order.
	names := make([]string, 0, 4)
	for _, candidate := range valueCandidates(Uint64(42)) {
		names = append(names, candidate.name)
	}
	want := []string{"I4", "UI8", "UI4", "BSTR"}
	if len(names) != len(want) {
		t.Fatalf("the candidates are %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("candidate %d is %q, want %q", i, names[i], want[i])
		}
	}
}

func TestValueCandidatesOfAText(t *testing.T) {
	// A text is only accepted as a string.
	candidates := valueCandidates(String("hello"))
	if len(candidates) != 1 || candidates[0].name != "BSTR" {
		t.Errorf("the candidates are %v, want only BSTR", candidates)
	}
}

func TestValueCandidatesOfALargeNumber(t *testing.T) {
	// A number that does not fit into a signed integer skips that
	// representation.
	candidates := valueCandidates(Uint64(0xFFFFFFFF))
	if candidates[0].name != "UI8" {
		t.Errorf("the first candidate is %q, want UI8", candidates[0].name)
	}
}
