//go:build windows

// Command wmiend validates the complete WMI method call chain using the
// vtable slot layout established from OpenPredator's Ghidra analysis of
// Acer's PSSvc.exe, cross-checked against measurements on this machine.
//
//	ConnectServer(3) -> GetObject(6) -> GetMethod(19)
//	  -> SpawnInstance(15) -> Put(5) -> ExecMethod(24) -> Get(4)
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

	// IWbemLocator::ConnectServer (slot 3), eight arguments.
	ns, _ := com.NewBSTR("root\\WMI")
	defer ns.Free()
	var services uintptr
	r, _ := locator.Call(3,
		uintptr(ns), // strNamespacePath
		0,           // strServer
		0,           // strLocale
		0,           // strUser
		0,           // strPassword
		0,           // lSecurityFlags
		0,           // strAuthority
		uintptr(unsafe.Pointer(&services)),
	)
	if int32(r) < 0 || services == 0 {
		fmt.Printf("ConnectServer failed 0x%08X\n", uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(services))
	fmt.Println("connected to root\\WMI")

	// IWbemServices::GetObject (slot 6).
	classPath, _ := com.NewBSTR("AcerGamingFunction")
	defer classPath.Free()
	var class, scratch uintptr
	r, _ = com.COMObject(services).Call(6,
		uintptr(classPath),
		0, // lFlags
		0, // ppCallResult
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)), // ppContext
	)
	fmt.Printf("GetObject -> 0x%08X class=%#x\n", uint32(r), class)
	if class == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(class))

	// IWbemClassObject::GetMethod (slot 19).
	methodName, _ := com.NewBSTR("GetGamingProfile")
	defer methodName.Free()
	var inDef, outDef uintptr
	r, _ = com.COMObject(class).Call(19,
		uintptr(methodName),
		0, // lFlags
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	fmt.Printf("GetMethod -> 0x%08X inDef=%#x outDef=%#x\n", uint32(r), inDef, outDef)
	if inDef == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(inDef))

	// IWbemClassObject::SpawnInstance (slot 15).
	var instance uintptr
	r, _ = com.COMObject(inDef).Call(15,
		0, // lFlags
		uintptr(unsafe.Pointer(&instance)),
	)
	fmt.Printf("SpawnInstance -> 0x%08X instance=%#x\n", uint32(r), instance)
	if r != 0 || instance == 0 {
		fmt.Println("SpawnInstance failed")
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(instance))

	// IWbemClassObject::Put (slot 5).
	inputName, _ := com.NewBSTR("gmInput")
	defer inputName.Free()
	input := com.NewVariantUI4(0)
	r, _ = com.COMObject(instance).Call(5,
		uintptr(inputName),
		0, // lFlags
		uintptr(unsafe.Pointer(&input)),
		0, // lType
	)
	fmt.Printf("Put(gmInput=0) -> 0x%08X\n", uint32(r))

	// Read it back with IWbemClassObject::Get (slot 4).
	{
		var back com.VARIANT
		back.Init()
		var flavor int32
		var ctx uintptr
		r, _ = com.COMObject(instance).Call(4,
			uintptr(inputName),
			0, // lFlags
			uintptr(unsafe.Pointer(&back)),
			uintptr(unsafe.Pointer(&flavor)),
			uintptr(unsafe.Pointer(&ctx)),
		)
		val, verr := back.AsUInt64()
		fmt.Printf("Get(gmInput) -> 0x%08X vt=0x%04X value=%d err=%v\n",
			uint32(r), back.Type(), val, verr)
		back.Clear()
	}

	// IWbemServices::ExecMethod (slot 24), seven arguments.
	instancePath, _ := com.NewBSTR(`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"`)
	defer instancePath.Free()
	var outParams uintptr
	var flagsOut int32
	r, _ = com.COMObject(services).Call(24,
		uintptr(instancePath),
		uintptr(methodName),
		0, // lFlags
		0, // ppCallResult
		uintptr(instance),
		uintptr(unsafe.Pointer(&outParams)),
		uintptr(unsafe.Pointer(&flagsOut)),
	)
	fmt.Printf("ExecMethod -> 0x%08X outParams=%#x\n", uint32(r), outParams)
	if int32(r) < 0 || outParams == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(outParams))

	// Read the output parameter.
	outName, _ := com.NewBSTR("gmOutput")
	defer outName.Free()
	var out com.VARIANT
	out.Init()
	var outFlavor int32
	var outCtx uintptr
	r, _ = com.COMObject(outParams).Call(4,
		uintptr(outName),
		0,
		uintptr(unsafe.Pointer(&out)),
		uintptr(unsafe.Pointer(&outFlavor)),
		uintptr(unsafe.Pointer(&outCtx)),
	)
	val, verr := out.AsUInt64()
	fmt.Printf("Get(gmOutput) -> 0x%08X vt=0x%04X value=%d (0x%X) err=%v\n",
		uint32(r), out.Type(), val, val, verr)
	out.Clear()
}
