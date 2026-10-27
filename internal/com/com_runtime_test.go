//go:build windows

package com

import (
	"testing"
)

// The COM runtime of the machine is there, so the calls that set it up
// and the ones that format a status code can be checked for real.

func TestInitCOM(t *testing.T) {
	if err := InitCOM(); err != nil {
		t.Fatalf("the COM runtime cannot be started: %v", err)
	}
	UninitCOM()
}

func TestInitCOMApartment(t *testing.T) {
	// The single threaded apartment is the one that the WMI locator
	// needs.
	if err := InitCOMApartment(ApartmentSingleThreaded); err != nil {
		t.Fatalf("the apartment cannot be started: %v", err)
	}
	UninitCOM()
}

func TestInitCOMSecurity(t *testing.T) {
	// The security of the COM runtime has to be established before the
	// first object is created, otherwise the WMI refuses the connection.
	if err := InitCOMSecurity(); err != nil {
		t.Fatalf("the security cannot be established: %v", err)
	}
}

func TestCoCreateInstanceOfTheLocator(t *testing.T) {
	if err := InitCOM(); err != nil {
		t.Fatalf("the COM runtime cannot be started: %v", err)
	}
	defer UninitCOM()
	if err := InitCOMSecurity(); err != nil {
		t.Fatalf("the security cannot be established: %v", err)
	}

	locator, err := CoCreateInstance(CLSIDWbemLocator, IIDWbemLocator, CLSCTXInprocServer)
	if err != nil {
		t.Fatalf("the locator cannot be created: %v", err)
	}
	defer locator.Release()

	if locator.IsNil() {
		t.Fatal("the locator is nil")
	}
	if !locator.Valid() {
		t.Error("the locator does not look like a live object")
	}
}

func TestCoCreateInstanceOfAClassThatIsNotThere(t *testing.T) {
	if err := InitCOM(); err != nil {
		t.Fatalf("the COM runtime cannot be started: %v", err)
	}
	defer UninitCOM()

	// A class that Windows does not know.
	unknown := GUID{
		Data1: 0x00000000, Data2: 0x0000, Data3: 0x0000,
		Data4: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
	}
	if _, err := CoCreateInstance(unknown, IIDWbemLocator, CLSCTXInprocServer); err == nil {
		t.Error("a class that is not there was created")
	}
}

func TestFormatHRESULT(t *testing.T) {
	// The status code of the WMI is not in the table of the system, the
	// text is taken from the library of the provider.
	text := FormatHRESULT(0x80041001)
	if text == "" {
		t.Fatal("the status code has no text")
	}
	// A code that no table knows still has to be rendered.
	text = FormatHRESULT(0x0000FFFF)
	if text == "" {
		t.Error("the status code has no text")
	}
}

func TestSetProxyBlanket(t *testing.T) {
	if err := InitCOM(); err != nil {
		t.Fatalf("the COM runtime cannot be started: %v", err)
	}
	defer UninitCOM()

	locator, err := CoCreateInstance(CLSIDWbemLocator, IIDWbemLocator, CLSCTXInprocServer)
	if err != nil {
		t.Fatalf("the locator cannot be created: %v", err)
	}
	defer locator.Release()

	// The blanket is set on every object of the WMI, the call must not
	// fail.
	SetProxyBlanket(locator)
}

func TestTheBSTR(t *testing.T) {
	bstr, err := NewBSTR("hello")
	if err != nil {
		t.Fatal(err)
	}
	defer bstr.Free()

	if bstr.IsNil() {
		t.Fatal("the string is nil")
	}
	if got := bstr.String(); got != "hello" {
		t.Errorf("the string is %q, want hello", got)
	}
}

func TestTheBSTROfAnEmptyText(t *testing.T) {
	bstr, err := NewBSTR("")
	if err != nil {
		t.Fatal(err)
	}
	defer bstr.Free()

	if got := bstr.String(); got != "" {
		t.Errorf("the string is %q, want an empty text", got)
	}
}

func TestTheBSTROfNothing(t *testing.T) {
	var bstr BSTR
	if !bstr.IsNil() {
		t.Error("the string is not nil")
	}
	if got := bstr.String(); got != "" {
		t.Errorf("the string is %q, want an empty text", got)
	}
	// Releasing nothing must not fail.
	bstr.Free()
}

func TestTheBSTROfATextWithAQuote(t *testing.T) {
	// The text of the WMI carries quotes and backslashes.
	bstr, err := NewBSTR(`a "quoted" \text\`)
	if err != nil {
		t.Fatal(err)
	}
	defer bstr.Free()

	want := `a "quoted" \text\`
	if got := bstr.String(); got != want {
		t.Errorf("the string is %q, want %q", got, want)
	}
}

func TestTheVariantOfTheEmptyType(t *testing.T) {
	var variant VARIANT
	variant.Init()
	defer variant.Clear()

	if variant.Type() != VTEmpty {
		t.Errorf("the variant has the type 0x%04X, want the empty one", variant.Type())
	}
}

func TestTheVariantThatIsClearedTwice(t *testing.T) {
	variant, err := NewVariantBSTR("hello")
	if err != nil {
		t.Fatal(err)
	}
	variant.Clear()
	// Clearing a variant that holds nothing must not fail.
	variant.Clear()
}

func TestTheVariantOfNothing(t *testing.T) {
	var variant *VARIANT
	variant.Clear()
}

func TestTheSafeArrayOfTheStrings(t *testing.T) {
	// The WMI returns the names of the properties as an array of
	// strings, so the array has to be read back the same way.
	names := []string{"Name", "Status", "Timeout"}
	array, err := NewSafeArrayBSTRs(names)
	if err != nil {
		t.Skipf("the array of strings cannot be created on this machine: %v", err)
	}
	defer array.Destroy()

	read, err := SafeArrayBSTRs(array.Pointer())
	if err != nil {
		t.Fatal(err)
	}
	if len(read) != len(names) {
		t.Fatalf("the array holds %v, want %v", read, names)
	}
	for i := range names {
		if read[i] != names[i] {
			t.Errorf("the name %d is %q, want %q", i, read[i], names[i])
		}
	}
}
