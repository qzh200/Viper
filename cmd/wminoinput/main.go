//go:build windows

// Command wminoinput isolates the ExecMethod signature.
//
// GetGamingFanTable takes no input parameters, so pInParams is NULL and
// the out-parameters object must land in a known argument slot.
package main

import (
	"fmt"
	"os"
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

	services, err := connect(locator, "root\\WMI")
	if err != nil {
		fmt.Println("ConnectServer:", err)
		os.Exit(1)
	}
	com.SetProxyBlanket(services)

	classPath, _ := com.NewBSTR("AcerGamingFunction")
	defer classPath.Free()
	var class, scratch uintptr
	r, _ := services.Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	fmt.Printf("GetObject -> 0x%08X class=%#x\n", uint32(r), class)
	if class == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(class))

	method, _ := com.NewBSTR("GetGamingFanTable")
	defer method.Free()
	var inDef, outDef uintptr
	r, _ = com.COMObject(class).Call(19,
		uintptr(method), 0,
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	fmt.Printf("GetMethod -> 0x%08X inDef=%#x outDef=%#x\n", uint32(r), inDef, outDef)

	paths := []string{
		`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"`,
		`AcerGamingFunction`,
	}

	for _, p := range paths {
		path, _ := com.NewBSTR(p)
		for _, slot := range []int{20, 28} {
			// Six argument shape with NULL in-parameters:
			//   path, method, flags, pInParams, ppOutParams, pFlags
			var v4, v5, v6 uintptr
			rr, _ := services.Call(slot,
				uintptr(path),
				uintptr(method),
				0,
				0, // pInParams = NULL
				uintptr(unsafe.Pointer(&v5)),
				uintptr(unsafe.Pointer(&v6)),
			)
			fmt.Printf("slot %d path=%s\n  -> 0x%08X v4=%#x v5=%#x v6=%#x\n", slot, p, uint32(rr), v4, v5, v6)
			os.Stdout.Sync()
			for _, cand := range []uintptr{v4, v5, v6} {
				if cand != 0 && com.COMObject(cand).Valid() {
					fmt.Printf("   out-params at %#x\n", cand)
					com.SetProxyBlanket(com.COMObject(cand))
					readOut(cand)
					return
				}
			}
		}
		path.Free()
	}
	fmt.Println("nothing worked")
}

// readOut reads the gmOutput parameter.
func readOut(outParams uintptr) {
	name, _ := com.NewBSTR("gmOutput")
	defer name.Free()
	var v com.VARIANT
	v.Init()
	var flavor int32
	var ctxOut uintptr
	r, _ := com.COMObject(outParams).Call(3,
		uintptr(name), 0,
		uintptr(unsafe.Pointer(&v)),
		uintptr(unsafe.Pointer(&flavor)),
		uintptr(unsafe.Pointer(&ctxOut)),
	)
	val, verr := v.AsUInt64()
	fmt.Printf("  Get(gmOutput) -> 0x%08X vt=0x%04X value=%d (0x%X) err=%v\n",
		uint32(r), v.Type(), val, val, verr)
	v.Clear()
}

// connect calls IWbemLocator::ConnectServer with the eight argument layout.
func connect(locator com.COMObject, ns string) (com.COMObject, error) {
	b, err := com.NewBSTR(ns)
	if err != nil {
		return 0, err
	}
	defer b.Free()

	var services uintptr
	r, err := locator.Call(3,
		uintptr(b),
		0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&services)),
	)
	if err != nil {
		return 0, err
	}
	if com.HRESULT(r).Failed() {
		return 0, fmt.Errorf("HRESULT 0x%08X", uint32(r))
	}
	if services == 0 {
		return 0, fmt.Errorf("S_OK but no services pointer")
	}
	return com.COMObject(services), nil
}
