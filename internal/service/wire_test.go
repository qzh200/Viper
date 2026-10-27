package service

import (
	"bytes"
	"testing"
)

func TestReadRequest(t *testing.T) {
	frame := []byte("ACER")
	frame = append(frame, 20, 0, 0, 0)
	frame = append(frame, `{"Function":"GPU_MODE"}`...)

	request, err := readRequest(bytes.NewReader(frame))
	if err != nil {
		t.Fatal(err)
	}
	if request.Packet != PacketGetUpdatedData {
		t.Errorf("the packet is %d, want %d", request.Packet, PacketGetUpdatedData)
	}
	if request.Function != "GPU_MODE" {
		t.Errorf("the function is %q, want GPU_MODE", request.Function)
	}
}

func TestReadRequestWithAParameter(t *testing.T) {
	frame := []byte("ACER")
	frame = append(frame, 100, 0, 0, 0)
	frame = append(frame, `{"Function":"BOOT_SOUND","Parameter":{"status":1}}`...)

	request, err := readRequest(bytes.NewReader(frame))
	if err != nil {
		t.Fatal(err)
	}
	if request.Packet != PacketSetDeviceData {
		t.Errorf("the packet is %d, want %d", request.Packet, PacketSetDeviceData)
	}
	parameter, ok := request.Parameter.(map[string]any)
	if !ok {
		t.Fatalf("the parameter is %T, want a set of values", request.Parameter)
	}
	value, err := number(parameter, "status")
	if err != nil {
		t.Fatal(err)
	}
	if value != 1 {
		t.Errorf("the status is %d, want 1", value)
	}
}

func TestReadRequestWithTheWrongWord(t *testing.T) {
	frame := []byte("NOPE")
	frame = append(frame, 20, 0, 0, 0)
	frame = append(frame, `{"Function":"GPU_MODE"}`...)

	if _, err := readRequest(bytes.NewReader(frame)); err == nil {
		t.Error("a request with the wrong word was accepted")
	}
}

func TestWriteResponse(t *testing.T) {
	var out bytes.Buffer
	if err := writeResponse(&out, "GPU_MODE", map[string]any{"mode": 1}); err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"data\" : \n\t{\n\t\t\"mode\" : 1\n\t},\n\t\"request\" : \"GPU_MODE\",\n\t\"result\" : \"0\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestWriteSetResponse(t *testing.T) {
	// The answer to a request that writes a setting carries no data.
	var out bytes.Buffer
	if err := writeSetResponse(&out, "BOOT_SOUND"); err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"request\" : \"BOOT_SOUND\",\n\t\"result\" : \"0\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestNumber(t *testing.T) {
	values := map[string]any{
		"mode":   float64(3),
		"status": "1",
		"text":   "not a number",
	}
	if value, err := number(values, "mode"); err != nil || value != 3 {
		t.Errorf("the mode is %d (%v), want 3", value, err)
	}
	if value, err := number(values, "status"); err != nil || value != 1 {
		t.Errorf("the status is %d (%v), want 1", value, err)
	}
	if _, err := number(values, "text"); err == nil {
		t.Error("a text was taken for a number")
	}
	if _, err := number(values, "missing"); err == nil {
		t.Error("a missing value was taken for a number")
	}
}

func TestParseColor(t *testing.T) {
	colour, ok := parseColor("#C70087")
	if !ok {
		t.Fatal("the colour was not accepted")
	}
	if colour.R != 0xC7 || colour.G != 0 || colour.B != 0x87 {
		t.Errorf("the colour is %+v, want the magenta", colour)
	}
	if _, ok := parseColor("nonsense"); ok {
		t.Error("a text was taken for a colour")
	}
}

func TestLightingEffectValue(t *testing.T) {
	cases := map[string]int{
		"STATIC":    0,
		"BREATHING": 1,
		"NEON":      2,
		"WAVE":      3,
		"SHIFTING":  4,
		"ZOOM":      5,
		"METEOR":    6,
		"TWINKLING": 7,
	}
	for name, want := range cases {
		got, ok := lightingEffectValue(name)
		if !ok {
			t.Errorf("the pattern %q was not accepted", name)
			continue
		}
		if int(got) != want {
			t.Errorf("the pattern %q is %d, want %d", name, got, want)
		}
	}
}
