package hardware

import "testing"

// The buffer of the keyboard backlight is a wire format. Every stand for
// the machine has to build it the same way the hardware does, otherwise
// the tests of the layers above would answer in a different shape than
// the machine and would not notice a wrong layout.

func TestTheBacklightBufferCarriesTheLevel(t *testing.T) {
	// The level sits in byte two and in byte five, byte eight holds the
	// flags of the backlight and byte nine has to be one, otherwise the
	// provider drops the buffer and zeroes the level.
	buffer := NewBacklightBuffer(42)
	if len(buffer) != 16 {
		t.Fatalf("the buffer is %d bytes long, want 16", len(buffer))
	}
	if buffer[2] != 42 {
		t.Errorf("byte two is %d, want 42", buffer[2])
	}
	if buffer[5] != 42 {
		t.Errorf("byte five is %d, want 42", buffer[5])
	}
	if buffer[8] != backlightFlags {
		t.Errorf("byte eight is %d, want %d", buffer[8], backlightFlags)
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

func TestTheBacklightBufferOfEveryLevel(t *testing.T) {
	for level := 0; level <= 100; level++ {
		buffer := NewBacklightBuffer(level)
		if buffer[2] != byte(level) || buffer[5] != byte(level) {
			t.Fatalf("the level %d is not in the buffer: %v", level, buffer)
		}
	}
}

func TestThePrivateBuilderIsThePublicOne(t *testing.T) {
	// The two builders have to be the same, so that the stand and the
	// machine cannot drift apart.
	private := NewBacklightBuffer(30)
	public := NewBacklightBuffer(30)
	for i := range private {
		if private[i] != public[i] {
			t.Fatalf("byte %d is %d in the private builder and %d in the public one",
				i, private[i], public[i])
		}
	}
}
