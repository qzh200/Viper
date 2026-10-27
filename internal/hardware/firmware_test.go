package hardware

import "testing"

// The firmware tables of the machine are read by the system, so the tests
// below run against the tables of the machine that runs them.

func TestSystemIdentity(t *testing.T) {
	model, family, product := SystemIdentity()
	if model == "" {
		t.Error("the machine has no product name")
	}
	if product == "" {
		t.Error("the machine has no product")
	}
	// The family is not carried by every machine, so it may be empty.
	_ = family
}

func TestSystemInfo(t *testing.T) {
	identity, err := SystemInfo()
	if err != nil {
		t.Fatal(err)
	}
	if identity.Manufacturer == "" {
		t.Error("the machine has no manufacturer")
	}
	if identity.Product == "" {
		t.Error("the machine has no product name")
	}
}

func TestIsPredator(t *testing.T) {
	// The call must not fail, the machine is either a Predator or not.
	if IsPredator() {
		identity, err := SystemInfo()
		if err != nil {
			t.Fatal(err)
		}
		if !IsPredatorModel(identity.Product, identity.Family) {
			t.Errorf("the machine %q of the family %q was taken for a Predator",
				identity.Product, identity.Family)
		}
	}
}

func TestProcessorClocks(t *testing.T) {
	speed, err := ProcessorClocks()
	if err != nil {
		t.Fatal(err)
	}
	if speed.MaxSpeedMHz == 0 {
		t.Error("the processor reports the highest clock as zero")
	}
	if speed.CurrentSpeedMHz == 0 {
		t.Error("the processor reports the clock as zero")
	}
}

func TestMemoryFrequency(t *testing.T) {
	// A machine without a memory module does not answer, a machine with
	// one reports a clock in megahertz.
	speed, err := MemoryFrequency()
	if err != nil {
		return
	}
	if speed <= 0 {
		t.Errorf("the memory reports the clock %d, want a number above zero", speed)
	}
}

func TestTheFirmwareTableIsAvailable(t *testing.T) {
	// The table is read by the system, it has to be there on a machine
	// that runs Windows.
	size := getSystemFirmwareTable(rsmb, 0, nil)
	if size == 0 {
		t.Fatal("the firmware table is not available")
	}
	if size < 8 {
		t.Fatalf("the firmware table is %d bytes long, want the header and the structures", size)
	}

	buffer := make([]byte, size)
	if getSystemFirmwareTable(rsmb, 0, buffer) == 0 {
		t.Fatal("the firmware table cannot be read")
	}
}
