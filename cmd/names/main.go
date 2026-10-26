//go:build windows

// Command names lists the properties of a WMI class object.
//
// It confirms whether the object returned by GetObject really is an
// IWbemClassObject, and shows the exact property names.
//
// Usage: names <ns> <class> [slot]
package main

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"viper/internal/com"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: names <ns> <class> [slot]")
		os.Exit(2)
	}
	namespace, className := os.Args[1], os.Args[2]
	slot := 13
	if len(os.Args) > 3 {
		slot, _ = strconv.Atoi(os.Args[3])
	}

	if err := com.InitCOM(); err != nil {
		fmt.Println("InitCOM:", err)
		os.Exit(1)
	}
	defer com.UninitCOM()
	if err := com.InitCOMSecurity(); err != nil {
		fmt.Println("InitCOMSecurity:", err)
	}

	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		fmt.Println("CoCreateInstance:", err)
		os.Exit(1)
	}
	defer locator.Release()

	b, _ := com.NewBSTR(namespace)
	defer b.Free()
	var services uintptr
	r, _ := locator.Call(3,
		uintptr(b),
		0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&services)),
	)
	if int32(r) < 0 || services == 0 {
		fmt.Printf("ConnectServer failed 0x%08X\n", uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(services))

	classPath, _ := com.NewBSTR(className)
	defer classPath.Free()
	var class, scratch uintptr
	r, _ = com.COMObject(services).Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	fmt.Printf("GetObject(%s) -> 0x%08X class=%#x\n", className, uint32(r), class)
	if class == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(class))

	// GetNames(long flags, SAFEARRAY **names, SAFEARRAY **qualifiers)
	var names, quals uintptr
	r, _ = com.COMObject(class).Call(slot,
		0,
		uintptr(unsafe.Pointer(&names)),
		uintptr(unsafe.Pointer(&quals)),
	)
	fmt.Printf("GetNames@%d -> 0x%08X names=%#x\n", slot, uint32(r), names)
	if names != 0 {
		list, err := com.SafeArrayBSTRs(names)
		fmt.Printf("  names (%v): %v\n", err, list)
	}
	if quals != 0 {
		com.COMObject(quals).Release()
	}
}
