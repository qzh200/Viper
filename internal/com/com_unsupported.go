//go:build !windows

// Package com is Windows-only. This stub keeps the module buildable and
// vet-clean on other platforms so that shared packages can be compiled and
// unit tested anywhere.
//
// 这个存根带着和 Windows 版一样的名字，所以调用者不必知道自己跑在哪个
// 平台上。凡是要碰 COM 运行时的入口都报 ErrUnsupported，纯粹的函数给出
// 它们不给 Windows 也能给出的答案。
package com

import (
	"errors"
	"fmt"
	"unsafe"
)

// ErrUnsupported is returned by every entry point on non-Windows systems.
var ErrUnsupported = errors.New("com: COM 和 WMI 这一层只在 Windows 上有")

// hresult converts a raw HRESULT bit pattern into the signed type.
func hresult(v uint32) HRESULT { return HRESULT(int32(v)) }

// GUID is a Windows COM class or interface identifier.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// String renders the GUID in the canonical registry form.
func (g GUID) String() string {
	return fmt.Sprintf("{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
		g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3],
		g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

// HRESULT is a COM status code.
type HRESULT int32

// Failed reports whether the status code indicates an error.
func (h HRESULT) Failed() bool { return int32(h) < 0 }

// Succeeded reports whether the status code indicates success.
func (h HRESULT) Succeeded() bool { return int32(h) >= 0 }

// Error implements the error interface.
func (h HRESULT) Error() string { return "HRESULT (这个平台不支持)" }

// The status codes of COM that the package uses.
var (
	// SOK is the successful status code.
	SOK = hresult(0x00000000)
	// SEAccessDenied is E_ACCESSDENIED.
	SEAccessDenied = hresult(0x80070005)
	// SEInvalidArg is E_INVALIDARG.
	SEInvalidArg = hresult(0x80070057)
	// SENotSupported is E_NOTIMPL.
	SENotSupported = hresult(0x8000FFFF)
	// SEInvalidClass is REGDB_E_CLASSNOTREG.
	SEInvalidClass = hresult(0x80040154)
	// SENoInterface is E_NOINTERFACE.
	SENoInterface = hresult(0x80004002)
	// SEFault is E_FAIL.
	SEFault = hresult(0x80004005)
	// SWbemErrorFailed is WBEM_E_FAILED.
	SWbemErrorFailed = hresult(0x80041001)
	// SWbemErrorNotFound is WBEM_E_NOT_FOUND.
	SWbemErrorNotFound = hresult(0x80041002)
	// SWbemErrorInvalidClass is WBEM_E_INVALID_CLASS.
	SWbemErrorInvalidClass = hresult(0x80041003)
	// SWbemErrorInvalidParameter is WBEM_E_INVALID_PARAMETER.
	SWbemErrorInvalidParameter = hresult(0x80041004)
	// SWbemErrorOutOfMemory is WBEM_E_OUT_OF_MEMORY.
	SWbemErrorOutOfMemory = hresult(0x80041006)
	// SWbemErrorInvalidQuery is WBEM_E_INVALID_QUERY.
	SWbemErrorInvalidQuery = hresult(0x80041017)
	// SWbemErrorMethodNotFound is WBEM_E_METHOD_NOT_FOUND.
	SWbemErrorMethodNotFound = hresult(0x8004101A)
	// SRPCEChangedMode is RPC_E_CHANGED_MODE.
	SRPCEChangedMode = hresult(0x80010106)
)

// The identifiers of the WMI repository.
var (
	// CLSIDWbemLocator is the class of the locator of WMI.
	CLSIDWbemLocator = GUID{}
	// IIDWbemLocator is the interface of the locator of WMI.
	IIDWbemLocator = GUID{}
)

// COMObject is a raw pointer to a COM interface.
type COMObject uintptr

// IsNil reports whether the object pointer is unset.
func (o COMObject) IsNil() bool { return o == 0 }

// Valid reports whether the pointer looks like a live COM object.
//
// Windows 之外没有 COM 的运行时，所以没有哪个指针是活的。
func (o COMObject) Valid() bool { return false }

// NewCOMObject always fails on non-Windows systems, because there is no
// address of an interface to wrap.
func NewCOMObject(ptr unsafe.Pointer) COMObject { return 0 }

// Call always fails on non-Windows systems.
func (o COMObject) Call(slot int, args ...uintptr) (uintptr, error) {
	return 0, ErrUnsupported
}

// Release is a no-op on non-Windows systems.
func (o COMObject) Release() {}

// FormatHRESULT renders a status code.
//
// Windows 的消息表不在这里，所以只给出编号。
func FormatHRESULT(hr uint32) string {
	return "HRESULT (这个平台不支持)"
}

// InitCOM always fails on non-Windows systems.
func InitCOM() error { return ErrUnsupported }

// InitCOMOwned always fails on non-Windows systems.
func InitCOMOwned() (bool, error) { return false, ErrUnsupported }

// InitCOMApartment always fails on non-Windows systems.
func InitCOMApartment(apartment uint32) error { return ErrUnsupported }

// UninitCOM is a no-op on non-Windows systems.
func UninitCOM() {}

// InitCOMSecurity always fails on non-Windows systems.
func InitCOMSecurity() error { return ErrUnsupported }

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
func NewVariantUI8(val uint64) VARIANT { return VARIANT{vt: VTUI8, data: val} }

// NewVariantUI4 builds a VT_UI4 variant.
func NewVariantUI4(val uint32) VARIANT { return VARIANT{vt: VTUI4, data: uint64(val)} }

// NewVariantI4 builds a VT_I4 variant.
func NewVariantI4(val int32) VARIANT { return VARIANT{vt: VTI4, data: uint64(uint32(val))} }

// NewVariantUI1 builds a VT_UI1 variant.
func NewVariantUI1(val uint8) VARIANT { return VARIANT{vt: VTUI1, data: uint64(val)} }

// NewVariantBool builds a VT_BOOL variant.
func NewVariantBool(val bool) VARIANT {
	var b uint64
	if val {
		b = 0xFFFFFFFF
	}
	return VARIANT{vt: VTBool, data: b}
}

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

// NewSafeArrayBSTRs always fails on non-Windows systems.
func NewSafeArrayBSTRs(texts []string) (SafeArray, error) { return SafeArray{}, ErrUnsupported }

// Destroy is a no-op on non-Windows systems.
func (s *SafeArray) Destroy() {}

// Bytes always fails on non-Windows systems.
func (s SafeArray) Bytes() ([]byte, error) { return nil, ErrUnsupported }

// SafeArrayBytes always fails on non-Windows systems.
func SafeArrayBytes(ptr uintptr) ([]byte, error) { return nil, ErrUnsupported }

// SafeArrayBSTRs always fails on non-Windows systems.
func SafeArrayBSTRs(ptr uintptr) ([]string, error) { return nil, ErrUnsupported }

// DestroySafeArray is a no-op on non-Windows systems.
func DestroySafeArray(ptr uintptr) {}
