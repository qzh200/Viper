package hardware

import (
	"encoding/hex"
	"testing"
)

func TestColorWord(t *testing.T) {
	// The word is blue, green and red from the top down and the zone in
	// the lowest byte. The zones are counted from one.
	cases := []struct {
		color RGBColor
		zone  uint64
		want  string
	}{
		{RGBColor{R: 199, G: 0, B: 135}, 1, "8700c701"},
		{RGBColor{R: 0, G: 174, B: 199}, 2, "c7ae0002"},
		{RGBColor{R: 255, G: 255, B: 255}, 4, "ffffff04"},
	}
	for _, c := range cases {
		got := hex.EncodeToString(uint64Bytes(colorWord(c.color, c.zone)))
		if got != c.want {
			t.Errorf("the colour word of %+v in zone %d is %s, want %s", c.color, c.zone, got, c.want)
		}
	}
}

func TestBacklightBuffer(t *testing.T) {
	// The level sits in byte two and in byte five, byte nine has to be one,
	// otherwise the provider drops the buffer and zeroes the level.
	buffer := backlightBuffer(42)
	if len(buffer) != 16 {
		t.Fatalf("the buffer is %d bytes long, want 16", len(buffer))
	}
	if buffer[2] != 42 {
		t.Errorf("byte two is %d, want 42", buffer[2])
	}
	if buffer[5] != 42 {
		t.Errorf("byte five is %d, want 42", buffer[5])
	}
	if buffer[8] != 3 {
		t.Errorf("byte eight is %d, want 3", buffer[8])
	}
	if buffer[9] != 1 {
		t.Errorf("byte nine is %d, want 1", buffer[9])
	}
	for i, b := range buffer {
		if i == 2 || i == 5 || i == 8 || i == 9 {
			continue
		}
		if b != 0 {
			t.Errorf("byte %d is %d, want 0", i, b)
		}
	}
}

func TestBacklightBufferLevelHundred(t *testing.T) {
	buffer := backlightBuffer(100)
	if buffer[2] != 100 || buffer[5] != 100 {
		t.Errorf("the buffer carries %d and %d, want 100 and 100", buffer[2], buffer[5])
	}
}

// uint64Bytes renders the lowest four bytes of a number, the most
// significant one first.
func uint64Bytes(value uint64) []byte {
	out := make([]byte, 4)
	for i := 0; i < 4; i++ {
		out[3-i] = byte(value >> (8 * i))
	}
	return out
}
