//go:build windows

// Command argprobe determines the real argument layout of
// IWbemLocator::ConnectServer by placing the namespace BSTR in each
// argument position and observing which one is accepted.
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

	nsBSTR, err := com.NewBSTR("root\\CIMV2")
	if err != nil {
		fmt.Println("NewBSTR:", err)
		return
	}
	defer nsBSTR.Free()

	fmt.Println("=== nine argument layout, BSTR at position i, ppLoc at 8 ===")
	for i := 0; i < 8; i++ {
		args := make([]uintptr, 9)
		args[i] = uintptr(nsBSTR)
		var services uintptr
		args[8] = uintptr(unsafe.Pointer(&services))

		r, _ := locator.Call(3, args...)
		corrupted := nsBSTR.String() != "root\\CIMV2"
		fmt.Printf("  pos %d -> HRESULT 0x%08X  services=%v  bstrCorrupted=%v\n",
			i, uint32(r), services != 0, corrupted)
		if services != 0 {
			com.COMObject(services).Release()
		}
	}

	fmt.Println("=== eight argument layout, BSTR at position i, out at 7 ===")
	for i := 0; i < 7; i++ {
		args := make([]uintptr, 8)
		args[i] = uintptr(nsBSTR)
		var services uintptr
		args[7] = uintptr(unsafe.Pointer(&services))

		r, _ := locator.Call(3, args...)
		corrupted := nsBSTR.String() != "root\\CIMV2"
		fmt.Printf("  pos %d -> HRESULT 0x%08X  services=%v  bstrCorrupted=%v\n",
			i, uint32(r), services != 0, corrupted)
		if services != 0 {
			com.COMObject(services).Release()
		}
	}

	fmt.Println("=== slot scan with all-NULL args (find a slot that answers) ===")
	for slot := 3; slot <= 6; slot++ {
		var services uintptr
		r, _ := locator.Call(slot,
			0, 0, 0, 0, 0, 0, 0, 0,
			uintptr(unsafe.Pointer(&services)),
		)
		fmt.Printf("  slot %d -> HRESULT 0x%08X  services=%v\n", slot, uint32(r), services != 0)
		if services != 0 {
			com.COMObject(services).Release()
		}
	}
}
