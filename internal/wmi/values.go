//go:build windows

package wmi

import (
	"fmt"
	"strconv"
	"unsafe"

	"viper/internal/com"
)

// slotGetNames is IWbemClassObject::GetNames.
const slotGetNames = 34

// failed reports whether a raw HRESULT indicates an error.
func failed(hr uintptr) bool { return int32(hr) < 0 }

// hresult renders a raw HRESULT for error messages.
func hresult(hr uintptr) error { return fmt.Errorf("HRESULT 0x%08X", uint32(hr)) }

// Kind describes the type of a WMI parameter value.
type Kind int

// The supported parameter kinds.
const (
	// KindUint64 is an unsigned 64 bit integer.
	KindUint64 Kind = iota
	// KindUint32 is an unsigned 32 bit integer.
	KindUint32
	// KindInt32 is a signed 32 bit integer.
	KindInt32
	// KindString is a string.
	KindString
	// KindBytes is a byte array, carried as an OLE SAFEARRAY.
	KindBytes
)

// Value is a WMI parameter value.
type Value struct {
	Kind  Kind
	U64   uint64
	Str   string
	Bytes []byte
}

// Uint64 returns an unsigned integer value.
func Uint64(v uint64) Value { return Value{Kind: KindUint64, U64: v} }

// Uint32 returns a 32 bit unsigned integer value.
func Uint32(v uint32) Value { return Value{Kind: KindUint32, U64: uint64(v)} }

// Int32 returns a signed integer value.
func Int32(v int32) Value { return Value{Kind: KindInt32, U64: uint64(uint32(v))} }

// String returns a string value.
func String(v string) Value { return Value{Kind: KindString, Str: v} }

// Bytes returns a byte array value.
func Bytes(v []byte) Value { return Value{Kind: KindBytes, Bytes: v} }

// String renders the value for display.
func (v Value) String() string {
	switch v.Kind {
	case KindString:
		return v.Str
	case KindBytes:
		return fmt.Sprintf("%d bytes", len(v.Bytes))
	default:
		return strconv.FormatUint(v.U64, 10)
	}
}

// UInt64 returns the value as an unsigned integer. Strings are parsed.
func (v Value) UInt64() uint64 {
	switch v.Kind {
	case KindString:
		n, _ := strconv.ParseUint(v.Str, 0, 64)
		return n
	case KindInt32:
		return uint64(uint32(v.U64))
	default:
		return v.U64
	}
}

// putValue writes an input parameter onto an in-parameters object.
//
// The Acer providers declare most parameters as unsigned integers but only
// accept a signed integer or a string when the value is stored. The
// preferred type is therefore tried first and the alternatives follow.
func putValue(target com.COMObject, name string, value Value) error {
	key, err := com.NewBSTR(name)
	if err != nil {
		return err
	}
	defer key.Free()

	if value.Kind == KindBytes {
		return putBytes(target, key, value.Bytes)
	}

	var lastErr error
	for _, candidate := range valueCandidates(value) {
		variant, err := candidate.build()
		if err != nil {
			lastErr = err
			continue
		}
		r, _ := target.Call(slotPut,
			uintptr(key),
			0, // lFlags
			uintptr(unsafe.Pointer(&variant)),
			0, // lType
		)
		variant.Clear()
		if failed(r) {
			lastErr = fmt.Errorf("wmi: Put(%s) as %s: %w", name, candidate.name, hresult(r))
			continue
		}
		return nil
	}
	return fmt.Errorf("wmi: Put(%s): no accepted type (%v)", name, lastErr)
}

// candidate is one attempt at writing a parameter.
type candidate struct {
	name  string
	build func() (com.VARIANT, error)
}

// valueCandidates lists the VARIANT representations to try for a value, in
// the order the Acer providers accept them.
func valueCandidates(value Value) []candidate {
	if value.Kind == KindString {
		return []candidate{
			{"BSTR", func() (com.VARIANT, error) { return com.NewVariantBSTR(value.Str) }},
		}
	}

	out := make([]candidate, 0, 4)
	if value.U64 <= 0x7FFFFFFF {
		// Signed 32 bit is what the providers actually store.
		out = append(out, candidate{"I4", func() (com.VARIANT, error) {
			return com.NewVariantI4(int32(value.U64)), nil
		}})
	}
	out = append(out, candidate{"UI8", func() (com.VARIANT, error) {
		return com.NewVariantUI8(value.U64), nil
	}})
	out = append(out, candidate{"UI4", func() (com.VARIANT, error) {
		return com.NewVariantUI4(uint32(value.U64)), nil
	}})
	out = append(out, candidate{"BSTR", func() (com.VARIANT, error) {
		return com.NewVariantBSTR(strconv.FormatUint(value.U64, 10))
	}})
	return out
}

// putBytes writes a byte array parameter using an OLE SAFEARRAY.
//
// SetGamingKBBacklight and SetGamingLED take their input as a byte array
// rather than an integer.
func putBytes(target com.COMObject, key com.BSTR, data []byte) error {
	variant, err := com.NewVariantUI1Array(data)
	if err != nil {
		return err
	}
	defer variant.Clear()

	r, _ := target.Call(slotPut,
		uintptr(key),
		0, // lFlags
		uintptr(unsafe.Pointer(&variant)),
		0, // lType
	)
	if failed(r) {
		return fmt.Errorf("wmi: Put(bytes): %w", hresult(r))
	}
	return nil
}

// readValue reads a named property from a class object.
func readValue(obj com.COMObject, name string) (Value, error) {
	key, err := com.NewBSTR(name)
	if err != nil {
		return Value{}, err
	}
	defer key.Free()

	var v com.VARIANT
	v.Init()
	defer v.Clear()

	var flavor int32
	r, _ := obj.Call(slotGet,
		uintptr(key),
		0, // lFlags
		uintptr(unsafe.Pointer(&v)),
		uintptr(unsafe.Pointer(&flavor)),
		0, // ppContext
	)
	if failed(r) {
		return Value{}, fmt.Errorf("wmi: Get(%s): %w", name, hresult(r))
	}

	switch v.Type() {
	case com.VTBSTR:
		s, err := v.AsString()
		if err != nil {
			return Value{}, err
		}
		return String(s), nil
	case com.VTArray | com.VTUI1:
		b, err := v.AsBytes()
		if err != nil {
			return Value{}, err
		}
		return Bytes(b), nil
	default:
		n, err := v.AsUInt64()
		if err != nil {
			return Value{}, err
		}
		return Uint64(n), nil
	}
}

// propertyNames lists the properties of a class object.
func propertyNames(obj com.COMObject) ([]string, error) {
	if obj == 0 {
		return nil, fmt.Errorf("wmi: GetNames: nil object")
	}
	var names, quals uintptr
	r, _ := obj.Call(slotGetNames,
		0, // lFlags
		uintptr(unsafe.Pointer(&names)),
		uintptr(unsafe.Pointer(&quals)),
	)
	if failed(r) {
		return nil, fmt.Errorf("wmi: GetNames: %w", hresult(r))
	}
	if names == 0 {
		return nil, fmt.Errorf("wmi: GetNames: no names")
	}
	if quals != 0 {
		com.COMObject(quals).Release()
	}
	return com.SafeArrayBSTRs(names)
}

// conventionalOutputNames are the output parameter names used by the WMI
// providers when the property list cannot be read.
//
// The Acer providers use gmOutput for the AcerGamingFunction class, uiOutput
// for APGeAction and gmReturn for the secondary value of the methods that
// return both a buffer and a status byte.
var conventionalOutputNames = []string{
	"gmOutput", "uiOutput", "gmReturn", "uiReturn",
	"ReturnValue", "Output", "Result",
}
