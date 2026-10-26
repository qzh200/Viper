//go:build windows

// Command wmiend2 validates the WMI method call chain.
//
//	ConnectServer(3) -> GetObject(6) -> GetMethod(19)
//	  -> SpawnInstance(15) -> Put(5) -> ExecMethod(24) -> Get(4)
//
// The method name and object path come from the command line so that
// several combinations can be tried without recompiling.
//
// Usage:
//
//	wmiend2 <ns> <class> <method> <pathVariant> <inputName> <outName>
package main

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"viper/internal/com"
)

func main() {
	if len(os.Args) < 7 {
		fmt.Println("usage: wmiend2 <ns> <class> <method> <pathVariant> <inputName> <outName>")
		os.Exit(2)
	}
	namespace, className, methodName := os.Args[1], os.Args[2], os.Args[3]
	pathVariant, _ := strconv.Atoi(os.Args[4])
	inputName, outName := os.Args[5], os.Args[6]

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

	ns, _ := com.NewBSTR(namespace)
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
	fmt.Printf("connected to %s\n", namespace)

	classPath, _ := com.NewBSTR(className)
	defer classPath.Free()
	var class uintptr
	r, _ = com.COMObject(services).Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		0,
	)
	fmt.Printf("GetObject -> 0x%08X class=%#x\n", uint32(r), class)
	if class == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(class))

	method, _ := com.NewBSTR(methodName)
	defer method.Free()
	var inDef, outDef uintptr
	r, _ = com.COMObject(class).Call(19,
		uintptr(method), 0,
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	fmt.Printf("GetMethod(%s) -> 0x%08X inDef=%#x outDef=%#x\n", methodName, uint32(r), inDef, outDef)
	if outDef == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(outDef))

	// Read the declared output parameter straight from the definition, so
	// that the property name and the Get slot can be validated.
	{
		name, _ := com.NewBSTR(outName)
		var v com.VARIANT
		v.Init()
		var flavor int32

		rr, _ := com.COMObject(outDef).Call(4,
			uintptr(name), 0,
			uintptr(unsafe.Pointer(&v)),
			uintptr(unsafe.Pointer(&flavor)),
			0,
		)
		val, verr := v.AsUInt64()
		fmt.Printf("outDef Get(%s) -> 0x%08X vt=0x%04X value=%d err=%v\n",
			outName, uint32(rr), v.Type(), val, verr)
		v.Clear()
	}

	// A method with no input parameters has no in-parameters object.
	var instance uintptr
	if inDef != 0 {
		com.SetProxyBlanket(com.COMObject(inDef))
		r, _ = com.COMObject(inDef).Call(15, 0, uintptr(unsafe.Pointer(&instance)), 0)
		fmt.Printf("SpawnInstance -> 0x%08X instance=%#x\n", uint32(r), instance)
		if r != 0 || instance == 0 {
			os.Exit(1)
		}
		com.SetProxyBlanket(com.COMObject(instance))

		name, _ := com.NewBSTR(inputName)
		defer name.Free()
		inputValue := uint32(0)
		if len(os.Args) > 8 {
			v, _ := strconv.ParseUint(os.Args[8], 0, 32)
			inputValue = uint32(v)
		}
		input := com.NewVariantI4(int32(inputValue))
		r, _ = com.COMObject(instance).Call(5,
			uintptr(name), 0,
			uintptr(unsafe.Pointer(&input)), 0,
		)
		fmt.Printf("Put(%s=0) -> 0x%08X\n", inputName, uint32(r))

		var back com.VARIANT
		back.Init()
		var bflavor int32

		r, _ = com.COMObject(instance).Call(4,
			uintptr(name), 0,
			uintptr(unsafe.Pointer(&back)),
			uintptr(unsafe.Pointer(&bflavor)),
			0,
		)
		val, verr := back.AsUInt64()
		fmt.Printf("Get(%s) -> 0x%08X vt=0x%04X value=%d err=%v\n",
			inputName, uint32(r), back.Type(), val, verr)
		back.Clear()
	}

	paths := []string{
		fmt.Sprintf(`%s.InstanceName="ACPI\\PNP0C14\\APGe_0"`, className),
		fmt.Sprintf(`%s.InstanceName="ACPI\PNP0C14\APGe_0"`, className),
		className,
		fmt.Sprintf(`\\.\%s:%s.InstanceName="ACPI\\PNP0C14\\APGe_0"`, namespace, className),
		fmt.Sprintf(`\\localhost\%s\%s.InstanceName="ACPI\\PNP0C14\\APGe_0"`, namespace, className),
		fmt.Sprintf(`\\localhost\%s\%s.InstanceName="ACPI\PNP0C14\APGe_0"`, namespace, className),
	}
	if pathVariant < 0 || pathVariant >= len(paths) {
		fmt.Println("path variant out of range")
		os.Exit(2)
	}
	path, _ := com.NewBSTR(paths[pathVariant])
	defer path.Free()
	fmt.Printf("path=%s\n", paths[pathVariant])

	var outParams uintptr

	slot := 24
	if len(os.Args) > 7 {
		slot, _ = strconv.Atoi(os.Args[7])
	}
	r, _ = com.COMObject(services).Call(slot,
		uintptr(path),
		uintptr(method),
		0, // lFlags
		0, // ppCallResult
		uintptr(instance),
		uintptr(unsafe.Pointer(&outParams)),
		0,
	)
	fmt.Printf("ExecMethod@%d -> 0x%08X outParams=%#x\n", slot, uint32(r), outParams)
	if int32(r) < 0 || outParams == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(outParams))

	name, _ := com.NewBSTR(outName)
	defer name.Free()
	var out com.VARIANT
	out.Init()
	var oflavor int32

	r, _ = com.COMObject(outParams).Call(4,
		uintptr(name), 0,
		uintptr(unsafe.Pointer(&out)),
		uintptr(unsafe.Pointer(&oflavor)),
		0,
	)
	val, verr := out.AsUInt64()
	fmt.Printf("Get(%s) -> 0x%08X vt=0x%04X value=%d (0x%X) err=%v\n",
		outName, uint32(r), out.Type(), val, val, verr)
	out.Clear()
}
