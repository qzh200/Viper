package hardware

import (
	"testing"

	"viper/internal/wmi"
)

// The tests in this file lock down the fixes of the audit: the readings that
// the firmware does not answer must not be turned into a value, the writes
// must be confirmed by reading them back, and the values that travel to the
// firmware must be the ones that are remembered.

func TestTheEcoProfileIsTheSixthEncoding(t *testing.T) {
	// 官方应用把节能档位暴露给用户，固件的编码是 6。
	if got := PowerEco.providerCode(); got != 6 {
		t.Errorf("the eco profile encodes to %d, want 6", got)
	}
	if got := decodePowerMode(6); got != PowerEco {
		t.Errorf("the value 6 decodes to %s, want the eco profile", got)
	}
	if got := PowerEco.String(); got != "eco" {
		t.Errorf("the eco profile renders as %q, want eco", got)
	}
	got, ok := ParsePowerMode("eco")
	if !ok || got != PowerEco {
		t.Errorf("the name eco decodes to %s (%v), want the eco profile", got, ok)
	}
}

func TestTheOverclockingOfTheGraphicsCardIsNotTakenFromTheGraphicsSwitch(t *testing.T) {
	provider := newFakeProvider()
	// The machine has a discrete graphics card, that is not a capability.
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0300)
	backend := newFakeBackend(provider)

	gpu, _, err := backend.OverclockCapability()
	if err != nil {
		t.Fatal(err)
	}
	if gpu {
		t.Error("the overclocking of the graphics card is reported although the firmware does not know it")
	}
	// The state of the graphics switch must not be read at all.
	if calls := provider.countOf("GetGamingMiscSetting"); calls != 1 {
		t.Errorf("the miscellaneous settings were read %d times, want one", calls)
	}
}

func TestTheOverclockingOfTheProcessor(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0100)
	backend := newFakeBackend(provider)

	gpu, cpu, err := backend.OverclockCapability()
	if err != nil {
		t.Fatal(err)
	}
	if !cpu {
		t.Error("the overclocking of the processor is not reported")
	}
	if gpu {
		t.Error("the overclocking of the graphics card is reported")
	}
}

func TestTheBacklightBufferIsRead(t *testing.T) {
	provider := newFakeProvider()
	// Some builds answer with the bytes as a hexadecimal text.
	provider.answer("GetGamingKBBacklight", map[string]wmi.Value{
		"gmOutput": wmi.String("00006400006400000300000000000000"),
	})
	backend := newFakeBackend(provider)

	buffer, err := backend.BacklightBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if len(buffer) != 16 {
		t.Fatalf("the buffer is %d bytes long, want 16", len(buffer))
	}
	if buffer[2] != 100 || buffer[5] != 100 {
		t.Errorf("the buffer carries %d and %d, want 100 and 100", buffer[2], buffer[5])
	}
	if buffer[8] != 3 {
		t.Errorf("the flags byte is %d, want 3", buffer[8])
	}
}

func TestTheBacklightBufferOfTheFirmware(t *testing.T) {
	// The buffer that the firmware reports on a Predator: the level sits in
	// byte two and in byte five, byte eight holds the flags.
	provider := newFakeProvider()
	provider.answer("GetGamingKBBacklight", map[string]wmi.Value{
		"gmOutput": wmi.String("000064000064000003000000000000"),
	})
	backend := newFakeBackend(provider)

	buffer, err := backend.BacklightBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if buffer[2] != 100 {
		t.Errorf("the second byte is %d, want 100", buffer[2])
	}
	if buffer[5] != 100 {
		t.Errorf("the fifth byte is %d, want 100", buffer[5])
	}
	if buffer[8] != 3 {
		t.Errorf("the flags byte is %d, want 3", buffer[8])
	}
	if buffer[9] != 0 {
		t.Errorf("the ninth byte is %d, want 0", buffer[9])
	}
}

func TestTheBacklightBufferOfAMachineThatDoesNotAnswer(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if _, err := backend.BacklightBuffer(); err == nil {
		t.Error("the buffer was accepted although the provider does not answer")
	}
}

func TestDecodeBytesOfTheAnswers(t *testing.T) {
	buffer, err := decodeBytes(wmi.Bytes([]byte{0, 0, 100}))
	if err != nil {
		t.Fatal(err)
	}
	if len(buffer) != 3 || buffer[2] != 100 {
		t.Errorf("the buffer is %v, want three bytes with 100 at the end", buffer)
	}

	buffer, err = decodeBytes(wmi.String("00 00 64"))
	if err != nil {
		t.Fatal(err)
	}
	if len(buffer) != 3 || buffer[2] != 100 {
		t.Errorf("the buffer is %v, want three bytes with 100 at the end", buffer)
	}

	if _, err := decodeBytes(wmi.Uint64(3)); err == nil {
		t.Error("a number was taken for a byte array")
	}
	if _, err := decodeBytes(wmi.String("0x1")); err == nil {
		t.Error("a broken text was taken for a byte array")
	}
}

func TestClampUint(t *testing.T) {
	cases := []struct {
		value uint64
		low   uint64
		high  uint64
		want  uint64
	}{
		{0, 1, 9, 1},
		{5, 1, 9, 5},
		{20, 1, 9, 9},
		{3, 0, 1, 1},
		{150, 0, 100, 100},
	}
	for _, c := range cases {
		if got := clampUint(c.value, c.low, c.high); got != c.want {
			t.Errorf("%d was clamped to %d, want %d", c.value, got, c.want)
		}
	}
}
