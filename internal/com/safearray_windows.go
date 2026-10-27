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

// safeArrayHead is the head of the descriptor of a SAFEARRAY.
//
// 这个布局是在这台机器的 64 位 Windows 上量出来的：维度的数目在第一个字的
// 低半段，接着是一个元素的大小，然后两个字是运行时自己留着的，再往后是
// 数据的地址和元素的个数。
type safeArrayHead struct {
	dims     uint32
	elemSize uint32
	features uint32
	locks    uint32
	data     uintptr
	elements uint32
}

// IsNil reports whether the array pointer is unset.
func (s SafeArray) IsNil() bool { return s.ptr == 0 }

// Pointer returns the raw SAFEARRAY pointer.
func (s SafeArray) Pointer() uintptr { return s.ptr }

// validSafeArray reports whether the address can be the address of a
// SAFEARRAY.
//
// 问不了 COM 的运行时：拿一个错的地址去调用，程序就结束了。所以这里直接
// 读描述符的头，把里面的字段检查一遍。装着别的东西的指针会被挡下来，这样
// 一次形状不对的调用就不会把程序带倒。
func validSafeArray(addr uintptr) bool {
	_, ok := safeArrayInfo(addr)
	return ok
}

// safeArrayInfo reads the head of the descriptor of an array.
//
// 第二个答案说明这个地址能不能是一个 SAFEARRAY 的地址。
func safeArrayInfo(addr uintptr) (safeArrayHead, bool) {
	var head safeArrayHead
	if addr < 0x10000 || addr >= 0x00007FFFFFFFFFFF {
		return head, false
	}
	head = *(*safeArrayHead)(pointerOf(addr))
	dims := head.dims & 0xFFFF
	if dims < 1 || dims > 3 {
		return head, false
	}
	if head.elemSize == 0 || head.elemSize > 64 {
		return head, false
	}
	if head.elements > 1<<20 {
		return head, false
	}
	if head.data < 0x10000 {
		return head, false
	}
	return head, true
}

// DestroySafeArray releases an array that was handed over as a raw
// address.
//
// 释放之前先检查地址：拿一个错的地址去调用，程序就结束了。
func DestroySafeArray(addr uintptr) {
	if !validSafeArray(addr) {
		return
	}
	procSafeArrayDestroy.Call(addr)
}

// NewSafeArrayUI1 creates a SAFEARRAY of VT_UI1 elements holding data.
//
// Acer 的 ACPI WMI 提供程序要的是键盘背光设置器收到一个固定的 16 字节的
// 缓冲区，而不是一个整数，这就是这个辅助函数存在的原因。
func NewSafeArrayUI1(data []byte) (SafeArray, error) {
	if len(data) == 0 {
		return SafeArray{}, fmt.Errorf("com: 不能建立空的 SAFEARRAY")
	}
	// SafeArrayCreateVector 返回的是 SAFEARRAY 的指针本身，不是状态码，
	// 所以只有空指针才表示失败。
	ptr, _, _ := procSafeArrayCreateVector.Call(
		uintptr(VTUI1),
		0, // 下界
		uintptr(len(data)),
	)
	if ptr == 0 {
		return SafeArray{}, fmt.Errorf("com: SafeArrayCreateVector 失败")
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
			return SafeArray{}, fmt.Errorf("com: SafeArrayPutElement(%d) 失败: HRESULT 0x%08X", i, uint32(hr2))
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
		return SafeArray{}, fmt.Errorf("com: 不能建立空的 SAFEARRAY")
	}
	ptr, _, _ := procSafeArrayCreateVector.Call(
		uintptr(VTBSTR),
		0, // 下界
		uintptr(len(texts)),
	)
	if ptr == 0 {
		return SafeArray{}, fmt.Errorf("com: SafeArrayCreateVector 失败")
	}
	arr := SafeArray{ptr: ptr}

	for i, text := range texts {
		idx := int32(i)
		bstr, err := NewBSTR(text)
		if err != nil {
			arr.Destroy()
			return SafeArray{}, err
		}
		// 字符串数组的元素就是字符串本身，所以交过去的是字符串的指针，
		// 不是装着它的那个变量的地址。
		hr, _, _ := procSafeArrayPutElement.Call(
			arr.ptr,
			uintptr(unsafe.Pointer(&idx)),
			uintptr(bstr),
		)
		// 数组已经拿着这个字符串了，调用者不能再放一次。
		bstr.Free()
		if HRESULT(hr).Failed() {
			arr.Destroy()
			return SafeArray{}, fmt.Errorf("com: SafeArrayPutElement(%d) 失败: HRESULT 0x%08X", i, uint32(hr))
		}
	}
	return arr, nil
}

// Destroy releases the SAFEARRAY.
//
// 数组的地址会被清掉，所以第二次调用不会把同一个数组再放一次。
func (s *SafeArray) Destroy() {
	if s == nil || s.ptr == 0 {
		return
	}
	procSafeArrayDestroy.Call(s.ptr)
	s.ptr = 0
}

// Bytes reads every element of a VT_UI1 SAFEARRAY.
func (s SafeArray) Bytes() ([]byte, error) {
	if s.ptr == 0 {
		return nil, fmt.Errorf("com: 空的 SAFEARRAY")
	}
	if !validSafeArray(s.ptr) {
		return nil, fmt.Errorf("com: 地址 %#x 不是一个 SAFEARRAY", s.ptr)
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
			return nil, fmt.Errorf("com: SafeArrayGetElement(%d) 失败: HRESULT 0x%08X", idx, hr2hex(hr))
		}
		out = append(out, val)
	}
	return out, nil
}

// SafeArrayBytes reads a VT_UI1 SAFEARRAY from a raw pointer.
func SafeArrayBytes(ptr uintptr) ([]byte, error) {
	if ptr == 0 {
		return nil, fmt.Errorf("com: 空的 SAFEARRAY")
	}
	if !validSafeArray(ptr) {
		return nil, fmt.Errorf("com: 地址 %#x 不是一个 SAFEARRAY", ptr)
	}
	return SafeArray{ptr: ptr}.Bytes()
}

// SafeArrayBSTRs reads a SAFEARRAY of BSTR elements from a raw pointer.
//
// WMI returns property name lists this way, so it is the way to discover
// which properties an object actually carries.
func SafeArrayBSTRs(ptr uintptr) ([]string, error) {
	if ptr == 0 {
		return nil, fmt.Errorf("com: 空的 SAFEARRAY")
	}
	head, ok := safeArrayInfo(ptr)
	if !ok {
		return nil, fmt.Errorf("com: 地址 %#x 不是一个 SAFEARRAY", ptr)
	}
	// 字符串数组的元素是一个指针。别的种类的数组会把字节当成字符串读出来，
	// 所以这里挡下来。
	if head.elemSize != uint32(ptrSize) {
		return nil, fmt.Errorf("com: SAFEARRAY %#x 里装的不是字符串", ptr)
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
			return nil, fmt.Errorf("com: SafeArrayGetElement(%d) 失败: HRESULT 0x%08X", idx, hr2hex(hr))
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
		1, // 维度
		uintptr(unsafe.Pointer(&lbound)),
	)
	if HRESULT(hr).Failed() {
		return 0, fmt.Errorf("com: SafeArrayGetLBound 失败: HRESULT 0x%08X", uint32(hr))
	}
	return lbound, nil
}

// getUBound returns the upper bound of the array's dimensions.
func (s SafeArray) getUBound() (int32, error) {
	var ubound int32
	hr, _, _ := procSafeArrayGetUBound.Call(
		s.ptr,
		1, // 维度
		uintptr(unsafe.Pointer(&ubound)),
	)
	if HRESULT(hr).Failed() {
		return 0, fmt.Errorf("com: SafeArrayGetUBound 失败: HRESULT 0x%08X", uint32(hr))
	}
	return ubound, nil
}

// hr2hex keeps HRESULT formatting in one place.
func hr2hex(hr uintptr) uint32 { return uint32(hr) }
