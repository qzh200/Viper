package hardware

import "testing"

func TestSystemIdentity(t *testing.T) {
	identity, err := SystemInfo()
	if err != nil {
		t.Skipf("the firmware tables cannot be read on this system: %v", err)
	}

	model, family, manufacturer := SystemIdentity()
	if model == "" {
		t.Error("the machine has no product name")
	}

	_ = family

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
