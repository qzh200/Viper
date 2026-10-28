package quickaccess

import (
	"bufio"
	"net"
	"testing"
)

func TestDisplayModeNames(t *testing.T) {
	cases := map[DisplayMode]string{
		DisplayModeHybrid:   "hybrid",
		DisplayModeDiscrete: "discrete",
		DisplayModeAuto:     "auto",
	}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("the mode %d renders as %q, want %q", mode, got, want)
		}
	}
	if got := DisplayMode(99).String(); got != "unknown" {
		t.Errorf("an unknown mode renders as %q, want unknown", got)
	}
}

func TestIntValueOfTheKinds(t *testing.T) {

	cases := []struct {
		value any
		want  int
	}{
		{float64(30), 30},
		{float64(30.0), 30},
		{30, 30},
		{"30", 30},
	}
	for _, c := range cases {
		got, err := intValue(Result{Value: c.value})
		if err != nil {
			t.Errorf("the value %v was refused: %v", c.value, err)
			continue
		}
		if got != c.want {
			t.Errorf("the value %v is %d, want %d", c.value, got, c.want)
		}
	}
}

func TestIntValueOfSomethingElse(t *testing.T) {
	cases := []any{true, nil, []any{1}, map[string]any{}}
	for _, value := range cases {
		if _, err := intValue(Result{Value: value}); err == nil {
			t.Errorf("the value %v was taken for a number", value)
		}
	}
}

func TestIntValueOfATextThatIsNotANumber(t *testing.T) {
	if _, err := intValue(Result{Value: "thirty"}); err == nil {
		t.Error("a text that is not a number was taken for a number")
	}
}

func TestSetDisplayModeRejectsTheOtherModes(t *testing.T) {
	client := &Client{}
	for _, mode := range []DisplayMode{0, 4, 99, -1} {
		if err := client.SetDisplayMode(mode); err == nil {
			t.Errorf("the mode %d was accepted", mode)
		}
	}
}

func TestSessionID(t *testing.T) {

	id := sessionID()
	if len(id) != 36 {
		t.Fatalf("the identifier is %q, want 36 characters", id)
	}
	for _, position := range []int{8, 13, 18, 23} {
		if id[position] != '-' {
			t.Errorf("the identifier %q does not carry a dash at %d", id, position)
		}
	}
	if id[14] != '4' {
		t.Errorf("the identifier %q is not of the fourth version", id)
	}

	switch id[19] {
	case '8', '9', 'a', 'b':
	default:
		t.Errorf("the identifier %q does not carry the variant of the fourth version", id)
	}

	if sessionID() == id {
		t.Error("two identifiers are the same")
	}
}

func TestFrameRoundTrip(t *testing.T) {

	cases := []string{
		"",
		"short",
		"A message that is longer than one hundred and twenty five characters, so that the length of the frame does not fit into a single byte.",
	}

	for _, text := range cases {
		sender, receiver := net.Pipe()
		defer sender.Close()
		defer receiver.Close()

		client := &Client{conn: sender, read: bufio.NewReader(sender)}

		done := make(chan error, 1)
		go func() {
			done <- client.writeFrame(opcodeText, []byte(text))
		}()

		reader := &Client{read: bufio.NewReader(receiver)}
		opcode, payload, err := reader.readFrame()
		if err != nil {
			t.Fatalf("the frame of %d characters cannot be read: %v", len(text), err)
		}
		if err := <-done; err != nil {
			t.Fatalf("the frame of %d characters cannot be written: %v", len(text), err)
		}
		if opcode != opcodeText {
			t.Errorf("the frame has the opcode %d, want %d", opcode, opcodeText)
		}
		if string(payload) != text {
			t.Errorf("the frame carries %q, want %q", payload, text)
		}
	}
}

func TestReadFrameOfALongMessage(t *testing.T) {

	text := make([]byte, 70000)
	for i := range text {
		text[i] = byte('a' + i%26)
	}

	sender, receiver := net.Pipe()
	defer sender.Close()
	defer receiver.Close()

	client := &Client{conn: sender, read: bufio.NewReader(sender)}

	done := make(chan error, 1)
	go func() {
		done <- client.writeFrame(opcodeText, text)
	}()

	reader := &Client{read: bufio.NewReader(receiver)}
	opcode, payload, err := reader.readFrame()
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if opcode != opcodeText {
		t.Errorf("the frame has the opcode %d, want %d", opcode, opcodeText)
	}
	if len(payload) != len(text) {
		t.Fatalf("the frame carries %d bytes, want %d", len(payload), len(text))
	}
	for i := range text {
		if payload[i] != text[i] {
			t.Fatalf("the byte %d of the frame is %d, want %d", i, payload[i], text[i])
		}
	}
}

func TestReadFrameOfAnUnmaskedMessage(t *testing.T) {

	frame := []byte{0x81, 0x05}
	frame = append(frame, "hello"...)

	sender, receiver := net.Pipe()
	defer sender.Close()
	defer receiver.Close()

	go func() {
		receiver.Write(frame)
	}()

	reader := &Client{read: bufio.NewReader(sender)}
	opcode, payload, err := reader.readFrame()
	if err != nil {
		t.Fatal(err)
	}
	if opcode != opcodeText {
		t.Errorf("the frame has the opcode %d, want %d", opcode, opcodeText)
	}
	if string(payload) != "hello" {
		t.Errorf("the frame carries %q, want hello", payload)
	}
}

func TestResultOfAnAnswerWithoutAValue(t *testing.T) {

	answer := map[string]any{
		"PacketType": float64(3),
		"Command":    "DarkMode",
		"Action":     "Get",
	}

	result, ok, err := resultOf(answer, "DarkMode")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("the answer was not accepted")
	}
	if result.Value != nil {
		t.Errorf("the value is %v, want nothing", result.Value)
	}
}

func TestResultOfAListOfValues(t *testing.T) {

	answer := map[string]any{
		"PacketType": float64(3),
		"Command":    "USBCharge",
		"Action":     "Get",
		"Result": []any{
			map[string]any{
				"Value":       float64(10),
				"Text":        "ten percent",
				"Description": "the first one",
			},
			map[string]any{
				"Value":       float64(30),
				"Text":        "thirty percent",
				"Description": "the second one",
			},
		},
	}

	result, ok, err := resultOf(answer, "USBCharge")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("the answer was not accepted")
	}
	value, isNumber := result.Value.(float64)
	if !isNumber || int(value) != 30 {
		t.Errorf("the value is %v, want 30", result.Value)
	}
	if result.Text != "thirty percent" {
		t.Errorf("the text is %q, want thirty percent", result.Text)
	}
}

func TestResultOfAPlainValue(t *testing.T) {

	answer := map[string]any{
		"PacketType": float64(3),
		"Command":    "USBChargeSwitch",
		"Action":     "Get",
		"Result":     true,
	}

	result, ok, err := resultOf(answer, "USBChargeSwitch")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("the answer was not accepted")
	}
	enabled, isSwitch := result.Value.(bool)
	if !isSwitch || !enabled {
		t.Errorf("the value is %v, want the switch on", result.Value)
	}
}

func TestResultOfAnotherCommand(t *testing.T) {

	answer := map[string]any{
		"PacketType": float64(3),
		"Command":    "DarkMode",
		"Action":     "Get",
	}

	if _, ok, _ := resultOf(answer, "USBCharge"); ok {
		t.Error("an answer of another command was accepted")
	}
}
