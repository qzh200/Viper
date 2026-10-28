package hardware

import "testing"

func TestContainsFold(t *testing.T) {
	cases := []struct {
		haystack string
		needle   string
		want     bool
	}{
		{"Predator PHN16-71", "predator", true},
		{"Predator PHN16-71", "PREDATOR", true},
		{"Predator PHN16-71", "Predator", true},
		{"Predator PHN16-71", "PHN16", true},
		{"Predator PHN16-71", "Nitro", false},
		{"Predator", "Predator PHN16-71", false},
		{"", "predator", false},
		{"Predator", "", false},
	}
	for _, c := range cases {
		if got := containsFold(c.haystack, c.needle); got != c.want {
			t.Errorf("the text %q holds %q: %v, want %v", c.haystack, c.needle, got, c.want)
		}
	}
}

func TestIndexFold(t *testing.T) {
	cases := []struct {
		haystack string
		needle   string
		want     int
	}{
		{"Predator PHN16-71", "predator", 0},
		{"Predator PHN16-71", "PHN", 9},
		{"Predator PHN16-71", "71", 15},
		{"Predator PHN16-71", "Nitro", -1},
		{"Predator", "", 0},
	}
	for _, c := range cases {
		if got := indexFold(c.haystack, c.needle); got != c.want {
			t.Errorf("the text %q holds %q at %d, want %d", c.haystack, c.needle, got, c.want)
		}
	}
}

func TestLower(t *testing.T) {
	cases := map[byte]byte{
		'A':  'a',
		'Z':  'z',
		'a':  'a',
		'0':  '0',
		'_':  '_',
		0x7F: 0x7F,
	}
	for input, want := range cases {
		if got := lower(input); got != want {
			t.Errorf("the byte %d becomes %d, want %d", input, got, want)
		}
	}
}

func TestCollectStringsOfAnEmptyTable(t *testing.T) {
	if got := collectStrings(nil); got != nil {
		t.Errorf("an empty table holds %v, want nothing", got)
	}
	if got := collectStrings([]byte{0x00, 0x00}); got != nil {
		t.Errorf("a table of two empty strings holds %v, want nothing", got)
	}
}

func TestCollectStringsStopsAtTheEmptyOne(t *testing.T) {

	table := []byte("Acer\x00\x00Predator\x00\x00")
	got := collectStrings(table)
	want := []string{"Acer"}
	if len(got) != len(want) {
		t.Fatalf("the table holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("string %d is %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPickOfAnEmptyTable(t *testing.T) {
	if got := pick(nil, 1); got != "" {
		t.Errorf("the first string of an empty table is %q, want an empty text", got)
	}
}

func TestNextStructureOfATruncatedTable(t *testing.T) {

	table := []byte{0x02, 0x08, 0x00, 0x00, 'x', 'y', 0x00}
	if got := nextStructure(table, 0, 8); got != len(table) {
		t.Errorf("the next structure starts at %d, want the end of the table at %d", got, len(table))
	}
}

func TestNextStructureWithoutAStringTable(t *testing.T) {

	table := []byte{0x02, 0x04, 0x00, 0x00, 0x00, 0x00, 0x01, 0x04}
	if got := nextStructure(table, 0, 4); got != 6 {
		t.Errorf("the next structure starts at %d, want 6", got)
	}
	if table[6] != 0x01 {
		t.Errorf("the next structure has the type %d, want 1", table[6])
	}
}

func TestParseIdentityOfATruncatedTable(t *testing.T) {

	table := []byte{0x01, 0x20, 0x00, 0x00}
	if _, ok := parseIdentity(table); ok {
		t.Error("a truncated structure was accepted")
	}
}

func TestParseIdentityOfAStructureWithoutStrings(t *testing.T) {

	table := []byte{0x01, 0x08, 0x00, 0x00, 0x01, 0x01, 0x01, 0x01}
	identity, ok := parseIdentity(table)
	if !ok {
		t.Fatal("the system information was not found")
	}
	if identity.Manufacturer != "" {
		t.Errorf("the manufacturer is %q, want an empty text", identity.Manufacturer)
	}
	if identity.Product != "" {
		t.Errorf("the product is %q, want an empty text", identity.Product)
	}
}

func TestParseIdentitySkipsTheOtherStructures(t *testing.T) {

	other := []byte{
		0x02, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		'x', 'y', 0x00, 0x00,
	}
	system := buildSystemTable()
	table := append(append([]byte{}, other...), system...)

	identity, ok := parseIdentity(table)
	if !ok {
		t.Fatal("the system information was not found")
	}
	if identity.Product != "Predator PHN16-71" {
		t.Errorf("the product is %q, want Predator PHN16-71", identity.Product)
	}
}
