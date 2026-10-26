//go:build windows

// Command wmidef compares the two accepted shapes of the in-parameters
// object for ExecMethod:
//
//	a) the in-parameters definition object itself
//	b) a spawned instance of the in-parameters class
//
// Both are filled with a correctly typed VARIANT before the call.
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
	instancePath, _ := com.NewBSTR(`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"`)
	defer instancePath.Free()

	// (a) the definition object
	tryCall(com.COMObject(services), inDef, inputName, instancePath, method, "definition")

	// (b) a spawned instance
	var instance uintptr
	r, _ = com.COMObject(inDef).Call(15, 0, uintptr(unsafe.Pointer(&instance)))
	fmt.Printf("SpawnInstance -> 0x%08X instance=%#x\n", uint32(r), instance)
	if r == 0 && instance != 0 {
		com.SetProxyBlanket(com.COMObject(instance))
		tryCall(com.COMObject(services), instance, inputName, instancePath, method, "spawned")
	}
}

// tryCall fills the input parameter on the given object and invokes the
// method with it.
func tryCall(services com.COMObject, inParams uintptr, name com.BSTR, path, method com.BSTR, label string) {
	value, _ := com.NewVariantBSTR("0")
	putRes, _ := com.COMObject(inParams).Call(5,
		uintptr(name), 0, uintptr(unsafe.Pointer(&value)), 0)
	fmt.Printf("%-12s Put -> 0x%08X\n", label, uint32(putRes))
	os.Stdout.Sync()

	var outParams uintptr
	var flagsOut int32
	r, _ := services.Call(24,
		uintptr(path),
		uintptr(method),
		0,
		0,
		uintptr(inParams),
		uintptr(unsafe.Pointer(&outParams)),
		uintptr(unsafe.Pointer(&flagsOut)),
	)
	fmt.Printf("%-12s ExecMethod -> 0x%08X outParams=%#x\n", label, uint32(r), outParams)
	os.Stdout.Sync()
	if int32(r) < 0 || outParams == 0 {
		return
	}
	com.SetProxyBlanket(com.COMObject(outParams))

	outName, _ := com.NewBSTR("gmOutput")
	defer outName.Free()
	var out com.VARIANT
	out.Init()
	var flavor int32
	var ctx uintptr
	r, _ = com.COMObject(outParams).Call(4,
		uintptr(outName), 0,
		uintptr(unsafe.Pointer(&out)),
		uintptr(unsafe.Pointer(&flavor)),
		uintptr(unsafe.Pointer(&ctx)),
	)
	val, verr := out.AsUInt64()
	fmt.Printf("%-12s Get(gmOutput) -> 0x%08X vt=0x%04X value=%d (0x%X) err=%v\n",
		label, uint32(r), out.Type(), val, val, verr)
	out.Clear()
}
