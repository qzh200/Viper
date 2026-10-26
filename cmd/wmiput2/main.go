//go:build windows

// Command wmiput2 compares the two ways of supplying the input parameters
// to ExecMethod:
//
//	a) Put straight onto the in-parameters definition object
//	b) Put onto a spawned instance of the in-parameters class
//
// The WMI provider rejects one of them, and the difference decides how the
// method call has to be written.
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

	method, _ := com.NewBSTR("GetGamingProfile")
	defer method.Free()
	var inDef, outDef uintptr
	r, _ = com.COMObject(class).Call(19,
		uintptr(method), 0,
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	fmt.Printf("GetMethod -> 0x%08X inDef=%#x\n", uint32(r), inDef)
	if inDef == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(inDef))

	inputName, _ := com.NewBSTR("gmInput")
	defer inputName.Free()

	// (a) Put straight onto the definition object.
	tryPut(inDef, inputName, "definition")

	// (b) Put onto a spawned instance.
	var instance uintptr
	r, _ = com.COMObject(inDef).Call(15, 0, uintptr(unsafe.Pointer(&instance)))
	fmt.Printf("SpawnInstance -> 0x%08X instance=%#x\n", uint32(r), instance)
	if r == 0 && instance != 0 {
		com.SetProxyBlanket(com.COMObject(instance))
		tryPut(instance, inputName, "spawned")
	}
}

// tryPut writes gmInput onto the given object and reads it back.
func tryPut(target uintptr, name com.BSTR, label string) {
	input := com.NewVariantUI4(7)
	putRes, _ := com.COMObject(target).Call(5,
		uintptr(name),
		0,
		uintptr(unsafe.Pointer(&input)),
		0,
	)
	fmt.Printf("%-12s Put -> 0x%08X\n", label, uint32(putRes))
	os.Stdout.Sync()

	var back com.VARIANT
	back.Init()
	var flavor int32
	var ctx uintptr
	getRes, _ := com.COMObject(target).Call(4,
		uintptr(name), 0,
		uintptr(unsafe.Pointer(&back)),
		uintptr(unsafe.Pointer(&flavor)),
		uintptr(unsafe.Pointer(&ctx)),
	)
	val, _ := back.AsUInt64()
	fmt.Printf("%-12s Get -> 0x%08X vt=0x%04X value=%d\n", label, uint32(getRes), back.Type(), val)
	os.Stdout.Sync()
	back.Clear()
}
