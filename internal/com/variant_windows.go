//go:build windows

package com

import (
	"fmt"
	"unsafe"
)

// VARIANT type codes (subset used by WMI).
const (
	VTEmpty    uint16 = 0
	VTNull     uint16 = 1
	VTI2       uint16 = 2
	VTI4       uint16 = 3
	VTR4       uint16 = 4
	VTR8       uint16 = 5
	VTCY       uint16 = 6
	VTDate     uint16 = 7
	VTBSTR     uint16 = 8
	VTDispatch uint16 = 9
	VTError    uint16 = 10
	VTBool     uint16 = 11
	VTObject   uint16 = 12
	VTUnknown  uint16 = 13
	VTI1       uint16 = 16
	VTUI1      uint16 = 17
	VTUI2      uint16 = 18
	VTUI4      uint16 = 19
	VTI8       uint16 = 20
	VTUI8      uint16 = 21
	VTInt      uint16 = 22
	VTUint     uint16 = 23
	VTVoid     uint16 = 24
	VTHRESULT  uint16 = 25
	VTArray    uint16 = 0x2000
)

// VARIANT is the standard COM automation union.
//
// Layout on both x86 and x64 is 16 bytes: a 2-byte type tag, three
// reserved WORDs, then an 8-byte union. We model the union as a uint64 so
// the struct size never depends on the pointer width.
type VARIANT struct {
	vt   uint16
	_    uint16
	_    uint16
	_    uint16
	data uint64
}

// Type returns the VARTYPE of the variant.
func (v *VARIANT) Type() uint16 { return v.vt }

// Clear releases any resources owned by the variant (BSTR, SAFEARRAY, ...)
// and resets it to VT_EMPTY.
func (v *VARIANT) Clear() {
	if v == nil {
		return
	}
	variantClear(v)
	*v = VARIANT{}
}

// Init prepares an empty variant.
func (v *VARIANT) Init() {
	*v = VARIANT{}
	variantInit(v)
}

// NewVariantUI8 builds a VT_UI8 variant.
func NewVariantUI8(val uint64) VARIANT {
	return VARIANT{vt: VTUI8, data: val}
}

// NewVariantUI4 builds a VT_UI4 variant.
func NewVariantUI4(val uint32) VARIANT {
	return VARIANT{vt: VTUI4, data: uint64(val)}
}

// NewVariantI4 builds a VT_I4 variant.
func NewVariantI4(val int32) VARIANT {
	return VARIANT{vt: VTI4, data: uint64(uint32(val))}
}

// NewVariantUI1 builds a VT_UI1 variant.
func NewVariantUI1(val uint8) VARIANT {
	return VARIANT{vt: VTUI1, data: uint64(val)}
}

// NewVariantBool builds a VT_BOOL variant.
func NewVariantBool(val bool) VARIANT {
	var b uint64
	if val {
		b = 0xFFFFFFFF
	}
	return VARIANT{vt: VTBool, data: b}
}

// NewVariantBSTR builds a VT_BSTR variant. The variant owns the string and
// releases it on Clear.
func NewVariantBSTR(s string) (VARIANT, error) {
	b, err := NewBSTR(s)
	if err != nil {
		return VARIANT{}, err
	}
	return VARIANT{vt: VTBSTR, data: uint64(uintptr(b))}, nil
}

// NewVariantUI1Array builds a VT_ARRAY|VT_UI1 variant holding a SAFEARRAY
// of bytes. The variant owns the array and destroys it on Clear.
func NewVariantUI1Array(data []byte) (VARIANT, error) {
	arr, err := NewSafeArrayUI1(data)
	if err != nil {
		return VARIANT{}, err
	}
	return VARIANT{vt: VTArray | VTUI1, data: uint64(arr.Pointer())}, nil
}

// AsUInt64 coerces the variant to an unsigned 64-bit value.
//
// The WMI providers return everything as an integer of some width, so we
// accept the integer family and fall back to a string parse, which some
// Acer BIOS MOFs use for numeric output.
func (v *VARIANT) AsUInt64() (uint64, error) {
	if v == nil {
		return 0, fmt.Errorf("com: nil variant")
	}

	switch v.vt {
	case VTUI8, VTUint, VTInt:
		return v.data, nil
	case VTUI4:
		return uint64(uint32(v.data)), nil
	case VTI4:
		return uint64(uint32(v.data)), nil
	case VTI8:
		return uint64(v.data), nil
	case VTUI2:
		return uint64(uint16(v.data)), nil
	case VTI2:
		return uint64(uint16(v.data)), nil
	case VTUI1:
		return v.data & 0xFF, nil
	case VTI1:
		return uint64(int8(v.data)), nil
	case VTBool:
		if v.data != 0 {
			return 1, nil
		}
		return 0, nil
	case VTBSTR:
		return parseBSTRToUint64(BSTR(uintptr(v.data)))
	}

	// Fall back to a coercion through oleaut32, which handles the exotic
	// cases (for example a VT_ARRAY of UI1).
	dst := VARIANT{}
	if hr := variantChangeType(&dst, v, VTUI8); hr.Succeeded() {
		out := dst.data
		dst.Clear()
		return out, nil
	}
	dst.Clear()
	return 0, fmt.Errorf("com: cannot convert variant of type 0x%04X to uint64", v.vt)
}

// AsBytes extracts the bytes of a VT_ARRAY|VT_UI1 variant.
func (v *VARIANT) AsBytes() ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("com: nil variant")
	}
	if v.vt&VTArray == 0 || v.vt&0xFF != VTUI1 {
		return nil, fmt.Errorf("com: variant is not a byte array (type 0x%04X)", v.vt)
	}
	return SafeArrayBytes(uintptr(v.data))
}

// AsString extracts a string from a VT_BSTR variant.
func (v *VARIANT) AsString() (string, error) {
	if v == nil {
		return "", fmt.Errorf("com: nil variant")
	}
	if v.vt != VTBSTR {
		return "", fmt.Errorf("com: variant is not a BSTR (type 0x%04X)", v.vt)
	}
	return BSTR(uintptr(v.data)).String(), nil
}

// parseBSTRToUint64 converts a BSTR holding a number into a uint64.
//
// Some Acer firmware MOFs declare numeric outputs as strings, so we accept
// both decimal and 0x-prefixed hexadecimal.
func parseBSTRToUint64(b BSTR) (uint64, error) {
	if b.IsNil() {
		return 0, fmt.Errorf("com: null BSTR")
	}
	s := trimSpace(b.String())
	if s == "" {
		return 0, nil
	}
	if len(s) > 2 && (s[0:2] == "0x" || s[0:2] == "0X") {
		var out uint64
		for _, c := range s[2:] {
			var d uint64
			switch {
			case c >= '0' && c <= '9':
				d = uint64(c - '0')
			case c >= 'a' && c <= 'f':
				d = uint64(c-'a') + 10
			case c >= 'A' && c <= 'F':
				d = uint64(c-'A') + 10
			default:
				return 0, fmt.Errorf("com: invalid hex BSTR %q", s)
			}
			out = out*16 + d
		}
		return out, nil
	}
	var out uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("com: invalid numeric BSTR %q", s)
		}
		out = out*10 + uint64(c-'0')
	}
	return out, nil
}

// trimSpace strips ASCII whitespace from both ends of s.
func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

// isSpace reports whether c is ASCII whitespace.
func isSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// variantPointer is a helper for passing a VARIANT by reference to COM.
func variantPointer(v *VARIANT) uintptr {
	return uintptr(unsafe.Pointer(v))
}
