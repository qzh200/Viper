//go:build windows

package wmi

import "testing"

// The status codes of COM are signed, a negative one is an error.

func TestFailed(t *testing.T) {
	cases := []struct {
		hr   uintptr
		want bool
	}{
		{0, false},
		{1, false},
		{0x80004005, true},
		{0x80041001, true},
	}
	for _, c := range cases {
		if got := failed(c.hr); got != c.want {
			t.Errorf("the status 0x%08X is a failure: %v, want %v", c.hr, got, c.want)
		}
	}
}

func TestHResult(t *testing.T) {
	err := hresult(0x80004005)
	if err == nil {
		t.Fatal("the status was not turned into an error")
	}
	if err.Error() != "HRESULT 0x80004005" {
		t.Errorf("the error reads %q, want HRESULT 0x80004005", err.Error())
	}
}

func TestParseNumeric(t *testing.T) {
	// The standard namespace reports some properties as a text, both
	// representations have to be accepted.
	cases := []struct {
		text string
		want uint64
	}{
		{"42", 42},
		{"0x2A", 42},
		{"0X2A", 42},
		{"  42  ", 42},
		{"", 0},
		{"   ", 0},
	}
	for _, c := range cases {
		got, err := parseNumeric(c.text)
		if err != nil {
			t.Errorf("the text %q was refused: %v", c.text, err)
			continue
		}
		if got != c.want {
			t.Errorf("the text %q is %d, want %d", c.text, got, c.want)
		}
	}
}

func TestParseNumericOfABrokenText(t *testing.T) {
	for _, text := range []string{"nonsense", "0x", "12.5"} {
		if _, err := parseNumeric(text); err == nil {
			t.Errorf("the text %q was taken for a number", text)
		}
	}
}

func TestTheConventionalOutputNames(t *testing.T) {
	// The property list of the out-parameters cannot be read on this
	// build, so the names of the providers are tried one by one.
	names := map[string]bool{}
	for _, name := range conventionalOutputNames {
		names[name] = true
	}
	for _, want := range []string{"gmOutput", "uiOutput", "gmReturn"} {
		if !names[want] {
			t.Errorf("the name %q is not tried, the conventional names are %v", want, conventionalOutputNames)
		}
	}
}

func TestTheKindsOfTheValues(t *testing.T) {
	// The kind tells the reader how the value has to be read.
	cases := []struct {
		value Value
		want  Kind
	}{
		{Uint64(1), KindUint64},
		{Uint32(1), KindUint32},
		{Int32(1), KindInt32},
		{String("1"), KindString},
		{Bytes([]byte{1}), KindBytes},
	}
	for _, c := range cases {
		if got := c.value.Kind; got != c.want {
			t.Errorf("the value %v has the kind %d, want %d", c.value, got, c.want)
		}
	}
}

func TestValueUInt64OfTheKinds(t *testing.T) {
	cases := []struct {
		value Value
		want  uint64
	}{
		{Uint64(0x6400), 0x6400},
		{Uint32(0x2A), 42},
		{Int32(-1), 0xFFFFFFFF},
		{Bytes([]byte{1, 2}), 0},
	}
	for _, c := range cases {
		if got := c.value.UInt64(); got != c.want {
			t.Errorf("the value %v is %d, want %d", c.value, got, c.want)
		}
	}
}

func TestValueStringOfTheKinds(t *testing.T) {
	cases := []struct {
		value Value
		want  string
	}{
		{Uint64(42), "42"},
		{Uint32(42), "42"},
		{String("hello"), "hello"},
		{Bytes([]byte{1, 2, 3}), "3 bytes"},
		{Bytes(nil), "0 bytes"},
	}
	for _, c := range cases {
		if got := c.value.String(); got != c.want {
			t.Errorf("the value %v renders as %q, want %q", c.value, got, c.want)
		}
	}
}
