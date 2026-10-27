package service

import (
	"testing"
)

// The answers of the original service, captured byte by byte. The writer of
// Viper has to produce exactly the same text, so that a client cannot tell
// the two services apart.
func TestWriteJSONMatchesTheOriginal(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{
			name: "the version",
			value: map[string]any{
				"data":    "1.2.2101.0",
				"request": "VERSION",
				"result":  "0",
			},
			want: "{\n\t\"data\" : \"1.2.2101.0\",\n\t\"request\" : \"VERSION\",\n\t\"result\" : \"0\"\n}\n",
		},
		{
			name: "the modes of the machine",
			value: map[string]any{
				"data":    []any{0, 1, 4, 5, 6},
				"request": "SUPPORT_OPERATING_MODE_CAPABILITY",
				"result":  "0",
			},
			want: "{\n\t\"data\" : \n\t[\n\t\t0,\n\t\t1,\n\t\t4,\n\t\t5,\n\t\t6\n\t],\n\t\"request\" : \"SUPPORT_OPERATING_MODE_CAPABILITY\",\n\t\"result\" : \"0\"\n}\n",
		},
		{
			name: "the empty answer",
			value: map[string]any{
				"data":    nil,
				"request": nil,
				"result":  "0",
			},
			want: "{\n\t\"data\" : null,\n\t\"request\" : null,\n\t\"result\" : \"0\"\n}\n",
		},
		{
			name: "a state of a setting",
			value: map[string]any{
				"data":    map[string]any{"status": 1},
				"request": "BACKLIGHT_30S",
				"result":  "0",
			},
			want: "{\n\t\"data\" : \n\t{\n\t\t\"status\" : 1\n\t},\n\t\"request\" : \"BACKLIGHT_30S\",\n\t\"result\" : \"0\"\n}\n",
		},
	}

	for _, c := range cases {
		if got := writeJSON(c.value); got != c.want {
			t.Errorf("%s\n got: %q\nwant: %q", c.name, got, c.want)
		}
	}
}

func TestWriteJSONSortsTheMembers(t *testing.T) {
	// The original service keeps the members of an object in the order of
	// their bytes.
	value := map[string]any{
		"zebra": 1,
		"alpha": 2,
		"LEDs":  3,
	}
	want := "{\n\t\"LEDs\" : 3,\n\t\"alpha\" : 2,\n\t\"zebra\" : 1\n}\n"
	if got := writeJSON(value); got != want {
		t.Errorf("the members are not in order\n got: %q\nwant: %q", got, want)
	}
}

func TestWriteJSONOfTheNumbers(t *testing.T) {
	cases := []struct {
		value any
		want  string
	}{
		{0.0, "0.0"},
		{5600.0, "5600.0"},
		{2487.5949510016339, "2487.5949510016339"},
		{45.01024060184006, "45.01024060184006"},
		{int64(42), "42"},
		{uint32(7), "7"},
	}
	for _, c := range cases {
		if got := writeJSON(c.value); got != c.want+"\n" {
			t.Errorf("the number %v is written as %q, want %q", c.value, got, c.want)
		}
	}
}

func TestWriteJSONOfTheEmptyThings(t *testing.T) {
	if got := writeJSON([]any{}); got != "[]\n" {
		t.Errorf("the empty array is written as %q", got)
	}
	if got := writeJSON(map[string]any{}); got != "{}\n" {
		t.Errorf("the empty object is written as %q", got)
	}
}

func TestWriteJSONOfTheTexts(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{"plain", "\"plain\""},
		{"with \"quotes\"", "\"with \\\"quotes\\\"\""},
		{"back\\slash", "\"back\\\\slash\""},
		{"line\nbreak", "\"line\\nbreak\""},
		{"中文", "\"中文\""},
	}
	for _, c := range cases {
		if got := writeJSON(c.value); got != c.want+"\n" {
			t.Errorf("the text %q is written as %q, want %q", c.value, got, c.want)
		}
	}
}
