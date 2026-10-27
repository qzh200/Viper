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

// uint64Bytes renders the lowest four bytes of a number, the most
// significant one first.
func uint64Bytes(value uint64) []byte {
	out := make([]byte, 4)
	for i := 0; i < 4; i++ {
		out[3-i] = byte(value >> (8 * i))
	}
	return out
}
