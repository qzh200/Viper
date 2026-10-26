//go:build !windows

// Package com is Windows-only. This stub keeps the module buildable and
// vet-clean on other platforms so that shared packages can be compiled and
// unit tested anywhere.
package com

import "errors"

// ErrUnsupported is returned by every entry point on non-Windows systems.
var ErrUnsupported = errors.New("com: the COM/WMI layer is only available on Windows")

// GUID is a Windows COM class or interface identifier.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// HRESULT is a COM status code.
type HRESULT int32

// Failed reports whether the status code represents an error.
func (h HRESULT) Failed() bool { return int32(h) < 0 }

// Succeeded reports whether the status code represents success.
func (h HRESULT) Succeeded() bool { return int32(h) >= 0 }

// Error implements the error interface.
func (h HRESULT) Error() string { return "HRESULT (unsupported platform)" }

// COMObject is a raw pointer to a COM interface.
type COMObject uintptr

// IsNil reports whether the object pointer is unset.
func (o COMObject) IsNil() bool { return o == 0 }

// Call always fails on non-Windows systems.
func (o COMObject) Call(slot int, args ...uintptr) (uintptr, error) {
	return 0, ErrUnsupported
}

// Release is a no-op on non-Windows systems.
func (o COMObject) Release() {}

// InitCOM always fails on non-Windows systems.
func InitCOM() error { return ErrUnsupported }

// UninitCOM is a no-op on non-Windows systems.
func UninitCOM() {}

// CoCreateInstance always fails on non-Windows systems.
func CoCreateInstance(clsid, iid GUID, ctx uint32) (COMObject, error) {
	return 0, ErrUnsupported
}

// SetProxyBlanket is a no-op on non-Windows systems.
func SetProxyBlanket(obj COMObject) {}

// BSTR is an allocated OLE string.
type BSTR uintptr

// IsNil reports whether the BSTR is unset.
func (b BSTR) IsNil() bool { return b == 0 }

// String always returns an empty string on non-Windows systems.
func (b BSTR) String() string { return "" }

// Free is a no-op on non-Windows systems.
func (b BSTR) Free() {}

// NewBSTR always fails on non-Windows systems.
func NewBSTR(s string) (BSTR, error) { return 0, ErrUnsupported }

// VARIANT is the standard COM automation union.
type VARIANT struct {
	vt   uint16
	_    uint16
	_    uint16
	_    uint16
	data uint64
}

// Type returns the VARTYPE of the variant.
func (v *VARIANT) Type() uint16 { return v.vt }

// Clear is a no-op on non-Windows systems.
func (v *VARIANT) Clear() {}

// Init is a no-op on non-Windows systems.
func (v *VARIANT) Init() {}

// NewVariantUI8 builds a VT_UI8 variant.
func NewVariantUI8(val uint64) VARIANT { return VARIANT{vt: 21, data: val} }

// NewVariantUI4 builds a VT_UI4 variant.
func NewVariantUI4(val uint32) VARIANT { return VARIANT{vt: 19, data: uint64(val)} }

// NewVariantI4 builds a VT_I4 variant.
func NewVariantI4(val int32) VARIANT { return VARIANT{vt: 3, data: uint64(uint32(val))} }

// NewVariantUI1 builds a VT_UI1 variant.
func NewVariantUI1(val uint8) VARIANT { return VARIANT{vt: 17, data: uint64(val)} }

// NewVariantBool builds a VT_BOOL variant.
func NewVariantBool(val bool) VARIANT { return VARIANT{vt: 11} }

// NewVariantBSTR always fails on non-Windows systems.
func NewVariantBSTR(s string) (VARIANT, error) { return VARIANT{}, ErrUnsupported }

// NewVariantUI1Array always fails on non-Windows systems.
func NewVariantUI1Array(data []byte) (VARIANT, error) { return VARIANT{}, ErrUnsupported }

// AsUInt64 always fails on non-Windows systems.
func (v *VARIANT) AsUInt64() (uint64, error) { return 0, ErrUnsupported }

// AsBytes always fails on non-Windows systems.
func (v *VARIANT) AsBytes() ([]byte, error) { return nil, ErrUnsupported }

// AsString always fails on non-Windows systems.
func (v *VARIANT) AsString() (string, error) { return "", ErrUnsupported }

// SafeArray describes the shape of an OLE SAFEARRAY.
type SafeArray struct {
	ptr uintptr
}

// IsNil reports whether the array pointer is unset.
func (s SafeArray) IsNil() bool { return s.ptr == 0 }

// Pointer returns the raw SAFEARRAY pointer.
func (s SafeArray) Pointer() uintptr { return s.ptr }

// NewSafeArrayUI1 always fails on non-Windows systems.
func NewSafeArrayUI1(data []byte) (SafeArray, error) { return SafeArray{}, ErrUnsupported }

// Destroy is a no-op on non-Windows systems.
func (s SafeArray) Destroy() {}

// Bytes always fails on non-Windows systems.
func (s SafeArray) Bytes() ([]byte, error) { return nil, ErrUnsupported }

// SafeArrayBytes always fails on non-Windows systems.
func SafeArrayBytes(ptr uintptr) ([]byte, error) { return nil, ErrUnsupported }
