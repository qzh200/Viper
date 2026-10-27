//go:build windows

package com

import (
	"fmt"
	"unsafe"
)

// SafeArray describes the shape of an OLE SAFEARRAY.
type SafeArray struct {
	ptr uintptr
}

// IsNil reports whether the array pointer is unset.
func (s SafeArray) IsNil() bool { return s.ptr == 0 }

// Pointer returns the raw SAFEARRAY pointer.
func (s SafeArray) Pointer() uintptr { return s.ptr }

// NewSafeArrayUI1 creates a SAFEARRAY of VT_UI1 elements holding data.
//
// Acer's ACPI WMI provider expects the keyboard backlight setter to receive
// a fixed 16-byte buffer rather than a plain integer, which is why this
// helper exists at all.
func NewSafeArrayUI1(data []byte) (SafeArray, error) {
	if len(data) == 0 {
		return SafeArray{}, fmt.Errorf("com: cannot create an empty SAFEARRAY")
	}
	// SafeArrayCreateVector returns the SAFEARRAY pointer itself, not a
	// status code, so only a null pointer means failure.
	ptr, _, _ := procSafeArrayCreateVector.Call(
		uintptr(VTUI1),
		0, // lower bound
		uintptr(len(data)),
	)
	if ptr == 0 {
		return SafeArray{}, fmt.Errorf("com: SafeArrayCreateVector failed")
	}
	arr := SafeArray{ptr: ptr}

	for i, b := range data {
		idx := int32(i)
		val := b
		hr2, _, _ := procSafeArrayPutElement.Call(
			arr.ptr,
			uintptr(unsafe.Pointer(&idx)),
			uintptr(unsafe.Pointer(&val)),
		)
		if HRESULT(hr2).Failed() {
			arr.Destroy()
			return SafeArray{}, fmt.Errorf("com: SafeArrayPutElement(%d) failed: HRESULT 0x%08X", i, uint32(hr2))
		}
	}
	return arr, nil
}

// NewSafeArrayBSTRs creates a SAFEARRAY of VT_BSTR elements holding the
// given texts.
//
// WMI returns the names of the properties this way, so the array has to
// be read back the same way.
func NewSafeArrayBSTRs(texts []string) (SafeArray, error) {
	if len(texts) == 0 {
		return SafeArray{}, fmt.Errorf("com: cannot create an empty SAFEARRAY")
	}
	ptr, _, _ := procSafeArrayCreateVector.Call(
		uintptr(VTBSTR),
		0, // lower bound
		uintptr(len(texts)),
	)
	if ptr == 0 {
		return SafeArray{}, fmt.Errorf("com: SafeArrayCreateVector failed")
	}
	arr := SafeArray{ptr: ptr}

	for i, text := range texts {
		idx := int32(i)
		bstr, err := NewBSTR(text)
		if err != nil {
			arr.Destroy()
			return SafeArray{}, err
		}
		// The element of an array of VT_BSTR is the string itself, so
		// the pointer of the string is handed over and not the address
		// of the variable that holds it.
		hr, _, _ := procSafeArrayPutElement.Call(
			arr.ptr,
			uintptr(unsafe.Pointer(&idx)),
			uintptr(bstr),
		)
		// The array holds the string now, the caller must not release it
		// a second time.
		bstr.Free()
		if HRESULT(hr).Failed() {
			arr.Destroy()
			return SafeArray{}, fmt.Errorf("com: SafeArrayPutElement(%d) failed: HRESULT 0x%08X", i, uint32(hr))
		}
	}
	return arr, nil
}

// Destroy releases the SAFEARRAY.
func (s SafeArray) Destroy() {
	if s.ptr != 0 {
		procSafeArrayDestroy.Call(s.ptr)
		s.ptr = 0
	}
}

// Bytes reads every element of a VT_UI1 SAFEARRAY.
func (s SafeArray) Bytes() ([]byte, error) {
	if s.ptr == 0 {
		return nil, fmt.Errorf("com: nil SAFEARRAY")
	}
	lbound, err := s.getLBound()
	if err != nil {
		return nil, err
	}
	ubound, err := s.getUBound()
	if err != nil {
		return nil, err
	}
	if ubound < lbound {
		return nil, nil
	}

	out := make([]byte, 0, ubound-lbound+1)
	for idx := lbound; idx <= ubound; idx++ {
		var val byte
		hr, _, _ := procSafeArrayGetElement.Call(
			s.ptr,
			uintptr(unsafe.Pointer(&idx)),
			uintptr(unsafe.Pointer(&val)),
		)
		if HRESULT(hr).Failed() {
			return nil, fmt.Errorf("com: SafeArrayGetElement(%d) failed: HRESULT 0x%08X", idx, uint32(hr2hex(hr)))
		}
		out = append(out, val)
	}
	return out, nil
}

// SafeArrayBytes reads a VT_UI1 SAFEARRAY from a raw pointer.
func SafeArrayBytes(ptr uintptr) ([]byte, error) {
	if ptr == 0 {
		return nil, fmt.Errorf("com: nil SAFEARRAY")
	}
	return SafeArray{ptr: ptr}.Bytes()
}

// SafeArrayBSTRs reads a SAFEARRAY of BSTR elements from a raw pointer.
//
// WMI returns property name lists this way, so it is the way to discover
// which properties an object actually carries.
func SafeArrayBSTRs(ptr uintptr) ([]string, error) {
	if ptr == 0 {
		return nil, fmt.Errorf("com: nil SAFEARRAY")
	}
	arr := SafeArray{ptr: ptr}
	lbound, err := arr.getLBound()
	if err != nil {
		return nil, err
	}
	ubound, err := arr.getUBound()
	if err != nil {
		return nil, err
	}
	if ubound < lbound {
		return nil, nil
	}

	out := make([]string, 0, ubound-lbound+1)
	for idx := lbound; idx <= ubound; idx++ {
		var b BSTR
		hr, _, _ := procSafeArrayGetElement.Call(
			arr.ptr,
			uintptr(unsafe.Pointer(&idx)),
			uintptr(unsafe.Pointer(&b)),
		)
		if HRESULT(hr).Failed() {
			return nil, fmt.Errorf("com: SafeArrayGetElement(%d) failed: HRESULT 0x%08X", idx, hr2hex(hr))
		}
		out = append(out, b.String())
		b.Free()
	}
	return out, nil
}

// getLBound returns the lower bound of the array's dimensions.
func (s SafeArray) getLBound() (int32, error) {
	var lbound int32
	hr, _, _ := procSafeArrayGetLBound.Call(
		s.ptr,
		1, // dimension
		uintptr(unsafe.Pointer(&lbound)),
	)
	if HRESULT(hr).Failed() {
		return 0, fmt.Errorf("com: SafeArrayGetLBound failed: HRESULT 0x%08X", uint32(hr))
	}
	return lbound, nil
}

// getUBound returns the upper bound of the array's dimensions.
func (s SafeArray) getUBound() (int32, error) {
	var ubound int32
	hr, _, _ := procSafeArrayGetUBound.Call(
		s.ptr,
		1, // dimension
		uintptr(unsafe.Pointer(&ubound)),
	)
	if HRESULT(hr).Failed() {
		return 0, fmt.Errorf("com: SafeArrayGetUBound failed: HRESULT 0x%08X", uint32(hr))
	}
	return ubound, nil
}

// hr2hex keeps HRESULT formatting in one place.
func hr2hex(hr uintptr) uint32 { return uint32(hr) }
