package service

import (
	"bytes"
	"math"
	"testing"
)

func TestWriteJSONOfTheBooleans(t *testing.T) {
	if got := writeJSON(true); got != "true\n" {
		t.Errorf("the switched on state is written as %q", got)
	}
	if got := writeJSON(false); got != "false\n" {
		t.Errorf("the switched off state is written as %q", got)
	}
}

func TestWriteJSONOfTheNestedThings(t *testing.T) {
	// A member that holds an object is written over several lines, the
	// original service does the same.
	value := map[string]any{
		"data": map[string]any{
			"mode": 1,
			"fans": []any{
				map[string]any{"name": "CPU"},
			},
		},
		"request": "FAN_CONTROL",
		"result":  "0",
	}
	want := "{\n\t\"data\" : \n\t{\n\t\t\"fans\" : \n\t\t[\n\t\t\t{\n\t\t\t\t\"name\" : \"CPU\"\n\t\t\t}\n\t\t],\n\t\t\"mode\" : 1\n\t},\n\t\"request\" : \"FAN_CONTROL\",\n\t\"result\" : \"0\"\n}\n"
	if got := writeJSON(value); got != want {
		t.Errorf("the document is\n%q\nwant\n%q", got, want)
	}
}

func TestWriteJSONOfAnEmptyMember(t *testing.T) {
	// An empty object and an empty array stay on one line, there is
	// nothing to spread over several lines.
	value := map[string]any{
		"list":  []any{},
		"table": map[string]any{},
	}
	want := "{\n\t\"list\" : [],\n\t\"table\" : {}\n}\n"
	if got := writeJSON(value); got != want {
		t.Errorf("the document is\n%q\nwant\n%q", got, want)
	}
}

func TestWriteJSONOfTheControlCharacters(t *testing.T) {
	// A character that has no place in a document is written as its code.
	cases := []struct {
		value string
		want  string
	}{
		{"\x01", "\"\\u0001\""},
		{"\x1f", "\"\\u001f\""},
		{"\b", "\"\\b\""},
		{"\f", "\"\\f\""},
		{"\r", "\"\\r\""},
		{"\t", "\"\\t\""},
	}
	for _, c := range cases {
		if got := writeJSON(c.value); got != c.want+"\n" {
			t.Errorf("the text %q is written as %q, want %q", c.value, got, c.want)
		}
	}
}

func TestWriteDoubleOfTheSpecialNumbers(t *testing.T) {
	// The original service writes a number that it cannot show as null.
	if got := writeDouble(math.NaN()); got != "null" {
		t.Errorf("the value that is not a number is written as %q, want null", got)
	}
	if got := writeDouble(math.Inf(1)); got != "null" {
		t.Errorf("the value that is too large is written as %q, want null", got)
	}
}

func TestWriteDoubleOfAWholeNumber(t *testing.T) {
	// A number without a fraction still shows one, the original service
	// prints seventeen significant digits.
	if got := writeDouble(42); got != "42.0" {
		t.Errorf("the whole number is written as %q, want 42.0", got)
	}
}

func TestIsComposite(t *testing.T) {
	cases := []struct {
		value any
		want  bool
	}{
		{[]any{1}, true},
		{[]any{}, false},
		{map[string]any{"a": 1}, true},
		{map[string]any{}, false},
		{42, false},
		{"text", false},
		{nil, false},
	}
	for _, c := range cases {
		if got := isComposite(c.value); got != c.want {
			t.Errorf("%v is composite: %v, want %v", c.value, got, c.want)
		}
	}
}

func TestReadJSONOfANestedDocument(t *testing.T) {
	// The end of the document is found by counting the braces, so a
	// document that holds objects inside objects is read as a whole.
	document := []byte(`{"Function":"LIGHTING","Parameter":{"LEDs":[{"color":"#C70087"}]}}`)

	body, err := readJSON(bytes.NewReader(document))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(document) {
		t.Errorf("the document is %q, want %q", body, document)
	}
}

func TestReadRequestWithoutADocument(t *testing.T) {
	// A frame that carries only the header is not a request. The reader
	// counts the braces of the document, so a frame without one ends in an
	// end of the stream and is refused.
	frame := []byte("ACER")
	frame = append(frame, 0, 0, 0, 0)

	if _, err := readRequest(bytes.NewReader(frame)); err == nil {
		t.Error("a frame without a document was accepted")
	}
}

func TestReadRequestOfATruncatedFrame(t *testing.T) {
	frame := []byte("ACER")
	frame = append(frame, 20, 0, 0)

	if _, err := readRequest(bytes.NewReader(frame)); err == nil {
		t.Error("a truncated frame was accepted")
	}
}

func TestReadRequestOfABrokenDocument(t *testing.T) {
	frame := []byte("ACER")
	frame = append(frame, 20, 0, 0, 0)
	frame = append(frame, `{"Function":`...)

	if _, err := readRequest(bytes.NewReader(frame)); err == nil {
		t.Error("a broken document was accepted")
	}
}

func TestWriteError(t *testing.T) {
	var out bytes.Buffer
	if err := writeError(&out, "GPU_MODE", "3"); err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"data\" : null,\n\t\"request\" : \"GPU_MODE\",\n\t\"result\" : \"3\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestWriteSetError(t *testing.T) {
	// The answer to a request that could not be carried out carries the
	// code of the failure and no data.
	var out bytes.Buffer
	if err := writeSetError(&out, "BOOT_SOUND", "4"); err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"request\" : \"BOOT_SOUND\",\n\t\"result\" : \"4\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestWriteEmptyResponse(t *testing.T) {
	// The original service answers without a name when it does not know
	// the request.
	var out bytes.Buffer
	if err := writeEmptyResponse(&out); err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"data\" : null,\n\t\"request\" : null,\n\t\"result\" : \"0\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}
