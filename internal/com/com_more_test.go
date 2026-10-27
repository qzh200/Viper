//go:build windows

package com

import (
	"testing"
)

// The layer of COM is thin, but the shapes of the values matter, because
// the WMI providers hand their answers back as VARIANTs.

func TestTheVariantTypes(t *testing.T) {
	cases := []struct {
		variant VARIANT
		want    uint16
	}{
		{NewVariantUI8(1), VTUI8},
		{NewVariantUI4(1), VTUI4},
		{NewVariantI4(1), VTI4},
		{NewVariantUI1(1), VTUI1},
		{NewVariantBool(true), VTBool},
	}
	for _, c := range cases {
		if got := c.variant.Type(); got != c.want {
			t.Errorf("the variant has the type 0x%04X, want 0x%04X", got, c.want)
		}
	}
}

func TestAsUInt64OfTheVariants(t *testing.T) {
	cases := []struct {
		variant VARIANT
		want    uint64
	}{
		{NewVariantUI8(0x6400), 0x6400},
		{NewVariantUI4(0x2A), 42},
		{NewVariantI4(-1), 0xFFFFFFFF},
		{NewVariantUI1(0xFF), 255},
		{NewVariantBool(true), 1},
		{NewVariantBool(false), 0},
	}
	for _, c := range cases {
		got, err := c.variant.AsUInt64()
		if err != nil {
			t.Errorf("the variant of the type 0x%04X was refused: %v", c.variant.Type(), err)
			continue
		}
		if got != c.want {
			t.Errorf("the variant of the type 0x%04X is %d, want %d", c.variant.Type(), got, c.want)
		}
	}
}

func TestAsUInt64OfNothing(t *testing.T) {
	var variant *VARIANT
	if _, err := variant.AsUInt64(); err == nil {
		t.Error("nothing was taken for a number")
	}
}

func TestAsStringOfAVariantThatIsNotAText(t *testing.T) {
	variant := NewVariantUI8(42)
	if _, err := variant.AsString(); err == nil {
		t.Error("a number was taken for a text")
	}
}

func TestAsBytesOfAVariantThatIsNotAnArray(t *testing.T) {
	variant := NewVariantUI8(42)
	if _, err := variant.AsBytes(); err == nil {
		t.Error("a number was taken for an array of bytes")
	}
}

func TestAsStringOfAText(t *testing.T) {
	variant, err := NewVariantBSTR("hello")
	if err != nil {
		t.Fatal(err)
	}
	defer variant.Clear()

	text, err := variant.AsString()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("the variant carries %q, want hello", text)
	}
}

func TestAsUInt64OfATextThatHoldsANumber(t *testing.T) {
	// Some of the firmware descriptions declare a number as a text.
	variant, err := NewVariantBSTR("0x2A")
	if err != nil {
		t.Fatal(err)
	}
	defer variant.Clear()

	got, err := variant.AsUInt64()
	if err != nil {
		t.Fatal(err)
	}
	if got != 42 {
		t.Errorf("the text is %d, want 42", got)
	}
}

func TestParseBSTRToUint64(t *testing.T) {
	cases := []struct {
		text string
		want uint64
	}{
		{"42", 42},
		{"0x2A", 42},
		{"0X2A", 42},
		{"  42  ", 42},
		{"", 0},
	}
	for _, c := range cases {
		bstr, err := NewBSTR(c.text)
		if err != nil {
			t.Fatal(err)
		}
		got, err := parseBSTRToUint64(bstr)
		bstr.Free()
		if err != nil {
			t.Errorf("the text %q was refused: %v", c.text, err)
			continue
		}
		if got != c.want {
			t.Errorf("the text %q is %d, want %d", c.text, got, c.want)
		}
	}
}

func TestParseBSTRToUint64OfABrokenText(t *testing.T) {
	for _, text := range []string{"nonsense", "0x", "12.5", "-1"} {
		bstr, err := NewBSTR(text)
		if err != nil {
			t.Fatal(err)
		}
		_, err = parseBSTRToUint64(bstr)
		bstr.Free()
		if err == nil {
			t.Errorf("the text %q was taken for a number", text)
		}
	}
}

func TestParseBSTRToUint64OfNothing(t *testing.T) {
	if _, err := parseBSTRToUint64(0); err == nil {
		t.Error("nothing was taken for a number")
	}
}

func TestTrimSpace(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"  hello  ", "hello"},
		{"\t\nhello\r\n", "hello"},
		{"hello", "hello"},
		{"   ", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := trimSpace(c.text); got != c.want {
			t.Errorf("the text %q becomes %q, want %q", c.text, got, c.want)
		}
	}
}

func TestIsSpace(t *testing.T) {
	for _, c := range []byte{' ', '\t', '\n', '\r', '\v', '\f'} {
		if !isSpace(c) {
			t.Errorf("the byte %d is not a space", c)
		}
	}
	for _, c := range []byte{'a', '0', 0x7F} {
		if isSpace(c) {
			t.Errorf("the byte %d is a space", c)
		}
	}
}

func TestTheSafeArrayOfBytes(t *testing.T) {
	// The keyboard backlight setter takes a buffer of sixteen bytes, so
	// the array has to carry every byte back.
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	array, err := NewSafeArrayUI1(data)
	if err != nil {
		t.Fatal(err)
	}
	defer array.Destroy()

	if array.IsNil() {
		t.Fatal("the array is nil")
	}
	if array.Pointer() == 0 {
		t.Error("the array has no pointer")
	}

	read, err := array.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(read) != len(data) {
		t.Fatalf("the array holds %d bytes, want %d", len(read), len(data))
	}
	for i := range data {
		if read[i] != data[i] {
			t.Errorf("the byte %d is %d, want %d", i, read[i], data[i])
		}
	}
}

func TestTheSafeArrayOfNothing(t *testing.T) {
	if _, err := NewSafeArrayUI1(nil); err == nil {
		t.Error("an empty array was created")
	}
}

func TestTheSafeArrayOfNothingRead(t *testing.T) {
	var array SafeArray
	if !array.IsNil() {
		t.Error("the array is not nil")
	}
	if _, err := array.Bytes(); err == nil {
		t.Error("nothing was read")
	}
	if _, err := SafeArrayBytes(0); err == nil {
		t.Error("nothing was read")
	}
	if _, err := SafeArrayBSTRs(0); err == nil {
		t.Error("nothing was read")
	}
}

func TestTheStatusCodes(t *testing.T) {
	// The status codes of COM are signed, the ones with the high bit set
	// are errors.
	if !SEAccessDenied.Failed() {
		t.Error("the access denied is not a failure")
	}
	if SEAccessDenied.Succeeded() {
		t.Error("the access denied is a success")
	}
	if SOK.Failed() {
		t.Error("the success is a failure")
	}
	if !SOK.Succeeded() {
		t.Error("the success is not a success")
	}
	if SEAccessDenied.Error() != "HRESULT 0x80070005" {
		t.Errorf("the error reads %q, want HRESULT 0x80070005", SEAccessDenied.Error())
	}
}

func TestTheGUID(t *testing.T) {
	guid := CLSIDWbemLocator
	want := "{4590F811-1D3A-11D0-891F-00AA004B2E24}"
	if got := guid.String(); got != want {
		t.Errorf("the identifier is %q, want %q", got, want)
	}
}

func TestTheCOMObject(t *testing.T) {
	var object COMObject
	if !object.IsNil() {
		t.Error("the object is not nil")
	}
	if object.Valid() {
		t.Error("the object is valid")
	}
	if _, err := object.Call(0); err == nil {
		t.Error("a call on nothing was accepted")
	}
}
