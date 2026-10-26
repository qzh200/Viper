//go:build windows

// Command validate checks which ConnectServer argument shape yields a real
// COM object, by verifying the result with IUnknown::AddRef.
package main

import (
	"fmt"
	"unsafe"

	"viper/internal/com"
)

func main() {
	if err := com.InitCOM(); err != nil {
		fmt.Println("InitCOM:", err)
		return
	}
	defer com.UninitCOM()
	if err := com.InitCOMSecurity(); err != nil {
		fmt.Println("InitCOMSecurity:", err)
	}

	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		fmt.Println("CoCreateInstance:", err)
		return
	}
	defer locator.Release()

	// Shape A: eight arguments, namespace first, out-pointer last.
	{
		b, _ := com.NewBSTR("root\\CIMV2")
		var out uintptr
		r, _ := locator.Call(3,
			uintptr(b), 0, 0, 0, 0, 0, 0,
			uintptr(unsafe.Pointer(&out)),
		)
		fmt.Printf("shape A (8 args) -> HRESULT 0x%08X out=%#x\n", uint32(r), out)
		validate(out)
		b.Free()
	}

	// Shape B: nine arguments, namespace second, out-pointer ninth.
	{
		b, _ := com.NewBSTR("root\\CIMV2")
		var out uintptr
		r, _ := locator.Call(3,
			0, uintptr(b), 0, 0, 0, 0, 0, 0,
			uintptr(unsafe.Pointer(&out)),
		)
		fmt.Printf("shape B (9 args) -> HRESULT 0x%08X out=%#x\n", uint32(r), out)
		validate(out)
		b.Free()
	}

	// Shape C: eight arguments, server first, namespace second, out last.
	{
		srv, _ := com.NewBSTR(".")
		ns, _ := com.NewBSTR("root\\CIMV2")
		var out uintptr
		r, _ := locator.Call(3,
			uintptr(srv), uintptr(ns), 0, 0, 0, 0, 0,
			uintptr(unsafe.Pointer(&out)),
		)
		fmt.Printf("shape C (8 args, srv+ns) -> HRESULT 0x%08X out=%#x\n", uint32(r), out)
		validate(out)
		srv.Free()
		ns.Free()
	}
}

// validate reports whether the pointer looks like a live COM object.
func validate(ptr uintptr) {
	if ptr == 0 {
		fmt.Println("    nil pointer")
		return
	}
	// Read the vtable pointer without calling anything.
	vtbl := *(*uintptr)(unsafe.Pointer(ptr))
	fmt.Printf("    vtable=%#x\n", vtbl)
	if vtbl == 0 {
		fmt.Println("    not a COM object (null vtable)")
		return
	}
	addRef, err := com.COMObject(ptr).Call(1)
	if err != nil {
		fmt.Println("    AddRef error:", err)
		return
	}
	fmt.Printf("    AddRef -> refcount=%d\n", addRef)
}
