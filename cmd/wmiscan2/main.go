//go:build windows

// Command wmiscan identifies the ExecMethod slot without risking a fault.
//
// A call with a method name that does not exist makes the WMI provider
// answer WBEM_E_METHOD_NOT_FOUND instead of reaching the hardware, so the
// right slot can be recognised by its error code while every wrong slot
// either errors differently or faults.
package main

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"viper/internal/com"
)

func main() {
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

	ns, _ := com.NewBSTR("root\\WMI")
	defer ns.Free()
	var services uintptr
	r, _ := locator.Call(3,
		uintptr(ns), 0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&services)),
	)
	if int32(r) < 0 || services == 0 {
		fmt.Printf("ConnectServer failed 0x%08X\n", uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(services))

	classPath, _ := com.NewBSTR("AcerGamingFunction")
	defer classPath.Free()
	var class, scratch uintptr
	r, _ = com.COMObject(services).Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	if class == 0 {
		fmt.Printf("GetObject failed 0x%08X\n", uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(class))

	// A method that certainly does not exist.
	method, _ := com.NewBSTR("NoSuchMethodZZ")
	defer method.Free()

	instancePath, _ := com.NewBSTR(`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"`)
	defer instancePath.Free()

	shapes := []struct {
		name string
		args func(out *uintptr, flags *int32) []uintptr
	}{
		{"7:path,method,flags,callRes,in,out,pFlags", func(out *uintptr, flags *int32) []uintptr {
			return []uintptr{uintptr(instancePath), uintptr(method), 0, 0, 0,
				uintptr(unsafe.Pointer(out)), uintptr(unsafe.Pointer(flags))}
		}},
		{"6:path,method,flags,in,out,pFlags", func(out *uintptr, flags *int32) []uintptr {
			return []uintptr{uintptr(instancePath), uintptr(method), 0, 0,
				uintptr(unsafe.Pointer(out)), uintptr(unsafe.Pointer(flags))}
		}},
		{"8:path,method,flags,callRes,in,out,pFlags,ctx", func(out *uintptr, flags *int32) []uintptr {
			return []uintptr{uintptr(instancePath), uintptr(method), 0, 0, 0,
				uintptr(unsafe.Pointer(out)), uintptr(unsafe.Pointer(flags)), 0}
		}},
	}

	first, last := 18, 40
	if len(os.Args) > 1 {
		first, _ = strconv.Atoi(os.Args[1])
		last = first
	}
	for slot := first; slot <= last; slot++ {
		for _, shape := range shapes {
			var out uintptr
			var flags int32
			r, _ := com.COMObject(services).Call(slot, shape.args(&out, &flags)...)
			fmt.Printf("slot %2d %-42s -> 0x%08X out=%#x\n", slot, shape.name, uint32(r), out)
			os.Stdout.Sync()
		}
	}
}
