package quickaccess

import (
	"encoding/base64"
	"testing"
)

func TestEncryptAESMatchesTheOriginal(t *testing.T) {
	// The answer of the original service, captured while the handshake
	// was running. The service sends the question back encrypted with the
	// second key, so the cipher has to produce exactly this text.
	const want = "g59El+p9nk0T6rt09R81GNLIQ3zCypACuJEjqudWGZGcIAGd7W7+tam+Lv4jdCnOScRIRXh8goNMXzZAuPjvMg=="

	sealed, err := encryptAES([]byte(question), []byte(keyB))
	if err != nil {
		t.Fatal(err)
	}
	got := base64.StdEncoding.EncodeToString(sealed)
	if got != want {
		t.Errorf("the sealed question is\n%s\nwant\n%s", got, want)
	}
}

func TestEncryptAESLength(t *testing.T) {
	// The cipher works in ECB mode, so the text is padded to whole blocks.
	for _, length := range []int{0, 1, 15, 16, 17, 45} {
		sealed, err := encryptAES(make([]byte, length), []byte(keyA))
		if err != nil {
			t.Fatal(err)
		}
		if len(sealed)%16 != 0 {
			t.Errorf("%d bytes produced %d sealed bytes, want a multiple of sixteen", length, len(sealed))
		}
		if len(sealed) < length {
			t.Errorf("%d bytes produced %d sealed bytes", length, len(sealed))
		}
	}
}

func TestAnswersCommand(t *testing.T) {
	cases := []struct {
		answer map[string]any
		want   bool
	}{
		{map[string]any{"PacketType": float64(3), "Command": "USBCharge"}, true},
		{map[string]any{"PacketType": float64(3), "Command": "BluelightShield"}, false},
		{map[string]any{"PacketType": float64(2), "Command": "USBCharge"}, false},
	}
	for _, c := range cases {
		if got := answersCommand(c.answer, "USBCharge"); got != c.want {
			t.Errorf("%v matches %v, want %v", c.answer, got, c.want)
		}
	}
}

func TestResultOf(t *testing.T) {
	answer := map[string]any{
		"PacketType": float64(3),
		"Command":    "USBCharge",
		"Action":     "Get",
		"Result": map[string]any{
			"Value": float64(30),
			"Text":  "Disable USB Charge when battery level under 30%",
		},
	}

	result, ok, err := resultOf(answer, "USBCharge")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("the answer was not accepted")
	}
	value, ok := result.Value.(float64)
	if !ok || int(value) != 30 {
		t.Errorf("the value is %v, want 30", result.Value)
	}
	if result.Text == "" {
		t.Error("the description is empty")
	}
}

func TestResultOfIgnoresNotifications(t *testing.T) {
	// The service sends notifications of its own beside the answers.
	answer := map[string]any{
		"PacketType": float64(3),
		"Command":    "USBCharge",
		"Action":     "Notify",
	}
	if _, ok, _ := resultOf(answer, "USBCharge"); ok {
		t.Error("a notification was taken for an answer")
	}
}

func TestBlueLightNames(t *testing.T) {
	cases := map[BlueLight]string{
		BlueLightDisabled:   "disabled",
		BlueLightLow:        "low",
		BlueLightMedium:     "medium",
		BlueLightHigh:       "high",
		BlueLightLongHour:   "long hour",
		BlueLightCustomised: "customised",
	}
	for level, want := range cases {
		if got := level.String(); got != want {
			t.Errorf("level %d renders as %q, want %q", level, got, want)
		}
	}
}

func TestSetUSBChargeLimitRejectsOtherValues(t *testing.T) {
	client := &Client{}
	for _, percent := range []int{0, 15, 25, 100} {
		if err := client.SetUSBChargeLimit(percent); err == nil {
			t.Errorf("the limit %d was accepted", percent)
		}
	}
}
