//go:build windows

// Package com provides a minimal, self-contained COM/OLE interop layer.
//
// The whole point of this package is that Viper must run as a single
// self-contained executable: no companion processes, no third-party DLLs
// (inpoutx64.dll, OpenRGB, ...). Everything here talks to ole32.dll and
// oleaut32.dll, which are part of Windows itself, through the standard
// library's lazy system DLL loader.
//
// It implements just enough of COM to drive WMI:
//   - CoInitializeEx / CoCreateInstance / CoSetProxyBlanket
//   - BSTR allocation (oleaut32)
//   - VARIANT construction and cleanup (oleaut32)
//   - SAFEARRAY construction (oleaut32)
//   - raw vtable dispatch for the IWbem* interfaces
package com

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

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

// HRESULT is a COM status code. Negative values indicate failure.
type HRESULT int32

// Failed reports whether the status code represents an error.
func (h HRESULT) Failed() bool { return int32(h) < 0 }

// Succeeded reports whether the status code represents success.
func (h HRESULT) Succeeded() bool { return int32(h) >= 0 }

// Error implements the error interface for failed status codes.
func (h HRESULT) Error() string {
	return fmt.Sprintf("HRESULT 0x%08X", uint32(h))
}

// hresult converts a raw HRESULT bit pattern into the signed type.
//
// This has to be a function rather than a constant expression: the WMI and
// Win32 error codes all have the high bit set, so they do not fit in an
// int32 literal without wrapping.
func hresult(v uint32) HRESULT { return HRESULT(int32(v)) }

// Common HRESULT values we care about.
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
)

// Well-known COM identifiers required to reach the WMI repository.
var (
	CLSIDWbemLocator = GUID{
		Data1: 0x4590F811, Data2: 0x1D3A, Data3: 0x11D0,
		Data4: [8]byte{0x89, 0x1F, 0x00, 0xAA, 0x00, 0x4B, 0x2E, 0x24},
	}
	IIDWbemLocator = GUID{
		Data1: 0xDC12A687, Data2: 0x737F, Data3: 0x11CF,
		Data4: [8]byte{0x88, 0x4D, 0x00, 0xAA, 0x00, 0x4B, 0x2E, 0x24},
	}
)

// Windows system DLLs. NewLazySystemDLL only searches the Windows system
// directories, so a rogue DLL placed next to the executable cannot hijack
// these imports.
var (
	ole32    = windows.NewLazySystemDLL("ole32.dll")
	oleaut32 = windows.NewLazySystemDLL("oleaut32.dll")

	procCoInitializeEx       = ole32.NewProc("CoInitializeEx")
	procCoCreateInstance     = ole32.NewProc("CoCreateInstance")
	procCoSetProxyBlanket    = ole32.NewProc("CoSetProxyBlanket")
	procCoUninitialize       = ole32.NewProc("CoUninitialize")
	procCoInitializeSecurity = ole32.NewProc("CoInitializeSecurity")

	procSysAllocString    = oleaut32.NewProc("SysAllocString")
	procSysFreeString     = oleaut32.NewProc("SysFreeString")
	procVariantInit       = oleaut32.NewProc("VariantInit")
	procVariantClear      = oleaut32.NewProc("VariantClear")
	procVariantChangeType = oleaut32.NewProc("VariantChangeType")

	procSafeArrayCreateVector = oleaut32.NewProc("SafeArrayCreateVector")
	procSafeArrayPutElement   = oleaut32.NewProc("SafeArrayPutElement")
	procSafeArrayGetElement   = oleaut32.NewProc("SafeArrayGetElement")
	procSafeArrayDestroy      = oleaut32.NewProc("SafeArrayDestroy")
	procSafeArrayGetLBound    = oleaut32.NewProc("SafeArrayGetLBound")
	procSafeArrayGetUBound    = oleaut32.NewProc("SafeArrayGetUBound")
)

// ptrSize is the native pointer width of the running process.
var ptrSize = unsafe.Sizeof(uintptr(0))

// COM apartment initialisation flags.
const (
	// ApartmentMultiThreaded selects a multi-threaded apartment.
	ApartmentMultiThreaded uint32 = coinitMultithreaded
	// ApartmentSingleThreaded selects a single-threaded apartment.
	ApartmentSingleThreaded uint32 = coinitApartment
)

// COM apartment initialisation flags.
const (
	coinitMultithreaded uint32 = 0x0
	coinitApartment     uint32 = 0x2
)

// CLSCTX flags.
const (
	// CLSCTXInprocServer requests an in-process COM server.
	CLSCTXInprocServer uint32 = 0x1
	// CLSCTXLocalServer requests a local out-of-process COM server.
	CLSCTXLocalServer uint32 = 0x4
)

// RPC impersonation levels used by CoSetProxyBlanket.
const (
	rpcAuthnLevelCall       uint32 = 3
	rpcImpnLevelImpersonate uint32 = 3
	rpcAuthnLevelWinNT      uint32 = 10
	rpcAuthzNone            uint32 = 0
)

// COMObject is a raw pointer to a COM interface. The zero value is invalid.
type COMObject uintptr

// IsNil reports whether the object pointer is unset.
func (o COMObject) IsNil() bool { return o == 0 }

// Valid reports whether the pointer looks like a live COM object.
//
// It is a cheap sanity check against out-parameters that a mis-shaped call
// may have filled with garbage. The vtable pointer has to be a plausible
// code address, not just a non-zero value.
func (o COMObject) Valid() bool {
	if o == 0 {
		return false
	}
	vtbl := *(*uintptr)(unsafe.Pointer(o))
	return vtbl != 0 && vtbl < 0x00007FFFFFFFFFFF
}

// vtblSlot resolves the address of the method at the given vtable index.
//
// A COM object is a pointer to a structure whose first member is the
// vtable pointer, so the vtable lives at *obj and slot N lives at
// vtable + N*ptrSize.
func (o COMObject) vtblSlot(slot int) uintptr {
	if o == 0 {
		return 0
	}
	vtbl := *(*uintptr)(unsafe.Pointer(o))
	if vtbl == 0 {
		return 0
	}
	return *(*uintptr)(unsafe.Pointer(vtbl + uintptr(slot)*ptrSize))
}

// Call dispatches to the method at the given vtable index.
func (o COMObject) Call(slot int, args ...uintptr) (uintptr, error) {
	fn := o.vtblSlot(slot)
	if fn == 0 {
		return 0, fmt.Errorf("com: vtable slot %d is nil", slot)
	}
	callArgs := make([]uintptr, 0, len(args)+1)
	callArgs = append(callArgs, uintptr(o))
	callArgs = append(callArgs, args...)
	r1, _, _ := syscall.SyscallN(fn, callArgs...)
	return r1, nil
}

// Release drops one reference on the object (IUnknown::Release, slot 2).
func (o COMObject) Release() {
	if o == 0 {
		return
	}
	_, _ = o.Call(2)
}

// procFormatMessage is used to decode WMI status codes.
var procFormatMessage = windows.NewLazySystemDLL("kernel32.dll").NewProc("FormatMessageW")

// FormatHRESULT renders a status code using the message table of the WMI
// provider, which is where the WBEM_E_* strings live.
//
// The WMI status codes are not in the system message table, so
// Win32Exception only reports "Unknown error". Loading fastprox.dll (and
// wbemprov.dll as a fallback) gives the real text.
func FormatHRESULT(hr uint32) string {
	const (
		formatMessageFromHMODULE   = 0x00000800
		formatMessageIgnoreInserts = 0x00001000
	)

	var mod uintptr
	for _, dll := range []string{"fastprox.dll", "wbemprov.dll"} {
		h, err := windows.LoadLibrary(dll)
		if err == nil {
			mod = uintptr(h)
			break
		}
	}

	buf := make([]uint16, 1024)
	flags := uintptr(formatMessageIgnoreInserts)
	if mod != 0 {
		flags |= formatMessageFromHMODULE
	}
	n, _, _ := procFormatMessage.Call(
		flags,
		mod,
		uintptr(hr),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
	)
	if n == 0 {
		return fmt.Sprintf("HRESULT 0x%08X", hr)
	}
	return strings.TrimSpace(windows.UTF16ToString(buf[:n]))
}

// InitCOM initialises the COM runtime for the current thread.
//
// The caller must invoke CoUninitialize exactly once for each successful
// InitCOM on the same thread. Apartment choice matters: WMI's IWbemLocator
// is apartment-threaded, so a single-threaded apartment is the safe
// default. A multi-threaded apartment is also available for callers that
// need it.
func InitCOM() error {
	return initCOM(coinitMultithreaded)
}

// InitCOMApartment initialises COM in the requested apartment.
func InitCOMApartment(apartment uint32) error {
	return initCOM(apartment)
}

func initCOM(apartment uint32) error {
	hr, _, _ := procCoInitializeEx.Call(0, uintptr(apartment), 0)
	h := HRESULT(hr)
	// S_FALSE (1) means the apartment was already initialized, which is fine.
	if h.Failed() && h != 1 {
		return fmt.Errorf("com: CoInitializeEx failed: %w", h)
	}
	return nil
}

// UninitCOM tears down the COM apartment for the current thread.
func UninitCOM() {
	procCoUninitialize.Call()
}

// CoCreateInstance instantiates a COM class.

// InitCOMSecurity establishes the process-wide COM security settings.
//
// The .NET System.Management wrapper (and therefore every WMI client that
// works through it) calls CoInitializeSecurity before touching the WMI
// repository. Skipping it can make IWbemLocator::ConnectServer fail with
// WBEM_E_INVALID_NAMESPACE.
func InitCOMSecurity() error {
	hr, _, _ := procCoInitializeSecurity.Call(
		0,           // pSecDesc
		^uintptr(0), // cAuthSvc = -1 (use the default set)
		0,           // asAuthSvc
		0,           // pReserved1
		uintptr(2),  // dwAuthnLevel = RPC_C_AUTHN_LEVEL_DEFAULT
		uintptr(3),  // dwImpLevel = RPC_C_IMP_LEVEL_IMPERSONATE
		0,           // pAuthList
		0,           // dwCapabilities
		0,           // rtl
	)
	h := HRESULT(hr)
	// RPC_E_TOO_LATE (0x80010119) means COM security was already set up,
	// which is perfectly acceptable. CO_E_SECURITY_INITIALIZATION
	// (0x800401F0) says the same thing in the wording of the older
	// builds: the security of the process is already in place and cannot
	// be changed a second time.
	if h.Failed() && h != hresult(0x80010119) && h != hresult(0x800401F0) {
		return fmt.Errorf("com: CoInitializeSecurity failed: %w", h)
	}
	return nil
}

func CoCreateInstance(clsid, iid GUID, ctx uint32) (COMObject, error) {
	var obj uintptr
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsid)),
		0,
		uintptr(ctx),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(&obj)),
	)
	h := HRESULT(hr)
	if h.Failed() {
		return 0, fmt.Errorf("com: CoCreateInstance(%s) failed: %w", clsid, h)
	}
	return COMObject(obj), nil
}

// SetProxyBlanket configures the RPC security context of a WMI proxy.
//
// Without this the WMI service rejects the call with WBEM_E_ACCESS_DENIED
// when running under a non-default impersonation level.
func SetProxyBlanket(obj COMObject) {
	procCoSetProxyBlanket.Call(
		uintptr(obj),
		uintptr(rpcAuthnLevelWinNT),
		uintptr(rpcAuthzNone),
		0,
		uintptr(rpcAuthnLevelCall),
		uintptr(rpcImpnLevelImpersonate),
		0,
		0,
	)
}

// BSTR is an allocated OLE string. The zero value is a null BSTR.
type BSTR uintptr

// IsNil reports whether the BSTR is unset.
func (b BSTR) IsNil() bool { return b == 0 }

// String converts the BSTR to a Go string and releases it.
func (b BSTR) String() string {
	if b == 0 {
		return ""
	}
	// A BSTR points at the first UTF-16 code unit; the length prefix sits
	// immediately before it. Reading the length keeps us from walking the
	// terminator, which is all we need for the null-terminated strings we
	// allocate ourselves.
	n := int(*(*uint32)(unsafe.Pointer(uintptr(b) - 4)))
	buf := unsafe.Slice((*uint16)(unsafe.Pointer(uintptr(b))), n)
	return windows.UTF16ToString(buf)
}

// Free releases the BSTR.
func (b BSTR) Free() {
	if b != 0 {
		procSysFreeString.Call(uintptr(b))
	}
}

// NewBSTR allocates an OLE string from a Go string.
//
// SysAllocString returns the BSTR pointer itself (not an HRESULT); a null
// return means the allocation failed.
func NewBSTR(s string) (BSTR, error) {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		return 0, err
	}
	r, _, _ := procSysAllocString.Call(uintptr(unsafe.Pointer(p)))
	if r == 0 {
		return 0, fmt.Errorf("com: SysAllocString failed for %d characters", len(s))
	}
	return BSTR(r), nil
}

// variantInit prepares a VARIANT for use.
func variantInit(v *VARIANT) {
	procVariantInit.Call(uintptr(unsafe.Pointer(v)))
}

// variantClear releases any resources held by a VARIANT.
func variantClear(v *VARIANT) {
	procVariantClear.Call(uintptr(unsafe.Pointer(v)))
}

// variantChangeType coerces a VARIANT into the requested type.
func variantChangeType(dst *VARIANT, src *VARIANT, vt uint16) HRESULT {
	hr, _, _ := procVariantChangeType.Call(
		uintptr(unsafe.Pointer(dst)),
		uintptr(unsafe.Pointer(src)),
		0,
		uintptr(vt),
	)
	return HRESULT(hr)
}

// errNoWMI is returned when a WMI class or method cannot be found.
var errNoWMI = errors.New("com: WMI object not found")
