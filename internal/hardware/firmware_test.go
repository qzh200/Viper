package hardware

import "testing"

// The firmware tables of the machine are read by the system, so the tests
// below run against the tables of the machine that runs them.
//
// A system that has no firmware tables answers with an error, and the test
// skips itself then. That keeps the suite green on the other systems
// without hiding a real failure on this one.

func TestSystemIdentity(t *testing.T) {
	identity, err := SystemInfo()
	if err != nil {
		t.Skipf("the firmware tables cannot be read on this system: %v", err)
	}

	model, family, manufacturer := SystemIdentity()
	if model == "" {
		t.Error("the machine has no product name")
	}
	// The family is not carried by every machine, so it may be empty.
	_ = family

	// The two ways of reading the identity have to agree. The three
	// values of SystemIdentity are the model, the family and the
	// manufacturer.
	if model != identity.Product {
		t.Errorf("the identity reports the model %q, want %q", model, identity.Product)
	}
	if manufacturer != identity.Manufacturer {
		t.Errorf("the identity reports the manufacturer %q, want %q", manufacturer, identity.Manufacturer)
	}
	if family != "" && family != identity.Family {
		t.Errorf("the identity reports the family %q, want %q", family, identity.Family)
	}
}

func TestSystemInfo(t *testing.T) {
	identity, err := SystemInfo()
	if err != nil {
		t.Skipf("the firmware tables cannot be read on this system: %v", err)
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
	if !IsPredator() {
		return
	}
	identity, err := SystemInfo()
	if err != nil {
		t.Fatalf("the machine is a Predator but its identity cannot be read: %v", err)
	}
	if !IsPredatorModel(identity.Product, identity.Family) {
		t.Errorf("the machine %q of the family %q was taken for a Predator",
			identity.Product, identity.Family)
	}
}

func TestTheFirmwareTableIsAvailable(t *testing.T) {
	// The table is read by the system, it has to be there on a machine
	// that runs Windows.
	size := getSystemFirmwareTable(rsmb, 0, nil)
	if size == 0 {
		t.Skip("the firmware table is not available on this system")
	}
	if size < 8 {
		t.Fatalf("the firmware table is %d bytes long, want the header and the structures", size)
	}

	buffer := make([]byte, size)
	if getSystemFirmwareTable(rsmb, 0, buffer) == 0 {
		t.Fatal("the firmware table cannot be read")
	}
}
