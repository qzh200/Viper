package nvidia

import "testing"

func TestReadWithoutTheDriver(t *testing.T) {
	// The machine of the test has no card of the vendor, the call has to
	// answer with an empty list and not with a panic.
	cards, err := Read()
	if err != nil {
		if !Available() {
			// The library is not there, that is the expected answer on a
			// machine without the driver.
			return
		}
		t.Fatal(err)
	}
	for _, card := range cards {
		if card.UsagePercent < 0 || card.UsagePercent > 100 {
			t.Errorf("the card reports a load of %d percent", card.UsagePercent)
		}
		if card.MaxGraphicsClockMHz < 0 {
			t.Errorf("the card reports the highest clock %d", card.MaxGraphicsClockMHz)
		}
	}
}

func TestIsIdle(t *testing.T) {
	if !(Telemetry{UsagePercent: 0}).IsIdle() {
		t.Error("a card without load is not idle")
	}
	if (Telemetry{UsagePercent: 3}).IsIdle() {
		t.Error("a card with load is idle")
	}
}

func TestFormatOfTheMemory(t *testing.T) {
	// The memory of a card is a number of bytes.
	telemetry := Telemetry{MemoryUsedBytes: 4 << 30, MemoryTotalBytes: 8 << 30}
	if telemetry.MemoryTotalBytes != 8<<30 {
		t.Errorf("the card holds %d bytes, want eight gigabytes", telemetry.MemoryTotalBytes)
	}
}
