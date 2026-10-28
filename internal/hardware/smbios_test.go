package hardware

import "testing"

func TestParseIdentity(t *testing.T) {

	table := buildSystemTable()

	identity, ok := parseIdentity(table)
	if !ok {
		t.Fatal("the system information was not found")
	}
	if identity.Manufacturer != "Acer" {
		t.Errorf("the manufacturer is %q, want Acer", identity.Manufacturer)
	}
	if identity.Product != "Predator PHN16-71" {
		t.Errorf("the product is %q, want Predator PHN16-71", identity.Product)
	}
	if identity.Version != "1.0" {
		t.Errorf("the version is %q, want 1.0", identity.Version)
	}
	if identity.SerialNumber != "SN12345" {
		t.Errorf("the serial number is %q, want SN12345", identity.SerialNumber)
	}
	if identity.Family != "Predator" {
		t.Errorf("the family is %q, want Predator", identity.Family)
	}
}

func TestParseIdentityWithoutFamily(t *testing.T) {

	table := buildSystemTable(0x1A)

	identity, ok := parseIdentity(table)
	if !ok {
		t.Fatal("the system information was not found")
	}
	if identity.Family != "" {
		t.Errorf("the family is %q, want an empty text", identity.Family)
	}
	if identity.Product != "Predator PHN16-71" {
		t.Errorf("the product is %q, want Predator PHN16-71", identity.Product)
	}
}

func TestParseIdentityWithoutStructure(t *testing.T) {
	if _, ok := parseIdentity([]byte{0x02, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); ok {
		t.Error("a table without a system information was accepted")
	}
}

func TestCollectStrings(t *testing.T) {
	table := []byte("Acer\x00Predator PHN16-71\x001.0\x00\x00")
	got := collectStrings(table)
	want := []string{"Acer", "Predator PHN16-71", "1.0"}
	if len(got) != len(want) {
		t.Fatalf("the table holds %d strings, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("string %d is %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPick(t *testing.T) {
	strings := []string{"Acer", "Predator"}
	if got := pick(strings, 1); got != "Acer" {
		t.Errorf("the first string is %q, want Acer", got)
	}
	if got := pick(strings, 2); got != "Predator" {
		t.Errorf("the second string is %q, want Predator", got)
	}
	if got := pick(strings, 0); got != "" {
		t.Errorf("the zero index returned %q, want an empty text", got)
	}
	if got := pick(strings, 9); got != "" {
		t.Errorf("an index behind the table returned %q, want an empty text", got)
	}
}

func TestNextStructure(t *testing.T) {

	table := []byte{
		0x02, 0x08, 0x00, 0x00, 'x', 'y', 0x00, 0x00,
		'a', 'b', 0x00, 0x00,
		0x01, 0x08,
	}
	next := nextStructure(table, 0, 8)
	if next != 12 {
		t.Errorf("the next structure starts at %d, want 12", next)
	}
	if table[next] != 0x01 {
		t.Errorf("the next structure has the type %d, want 1", table[next])
	}
}

func buildSystemTable(length ...int) []byte {
	size := 0x1B
	if len(length) > 0 {
		size = length[0]
	}

	structure := make([]byte, size)
	structure[0] = systemInformationType
	structure[1] = byte(size)
	structure[2] = 0x01
	structure[3] = 0x00
	structure[4] = 1
	structure[5] = 2
	structure[6] = 3
	structure[7] = 4
	if size >= 0x1B {
		structure[0x1A] = 5
	}

	strings := []byte("Acer\x00Predator PHN16-71\x001.0\x00SN12345\x00")
	if size >= 0x1B {
		strings = append(strings, "Predator\x00"...)
	}
	strings = append(strings, 0x00)

	out := make([]byte, 0, len(structure)+len(strings))
	out = append(out, structure...)
	out = append(out, strings...)
	return out
}
