//go:build windows

// Command wmifinal attempts one specific combination of a WMI method call.
//
// Every degree of freedom is a command line argument so that a fault in
// one combination cannot hide the others.
//
// Usage:
//
//	wmifinal <ns> <class> <method> <spawnSlot> <pathVariant> <outParam>
//
// pathVariant is an index into the candidate object paths.
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
		fmt.Println("usage: wmifinal <ns> <class> <method> <spawnSlot> <pathVariant> <outParam>")
		os.Exit(2)
	}
	namespace, className, methodName := os.Args[1], os.Args[2], os.Args[3]
	spawnSlot, _ := strconv.Atoi(os.Args[4])
	pathVariant, _ := strconv.Atoi(os.Args[5])
	outParam := os.Args[6]

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

	services, err := connect(locator, namespace)
	if err != nil {
		fmt.Println("ConnectServer:", err)
		os.Exit(1)
	}
	com.SetProxyBlanket(services)

	classPath, _ := com.NewBSTR(className)
	defer classPath.Free()
	var class, scratch uintptr
	r, _ := services.Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	if class == 0 {
		fmt.Printf("GetObject failed 0x%08X\n", uint32(r))
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
	if inDef == 0 {
		fmt.Printf("GetMethod failed 0x%08X\n", uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(inDef))

	// Spawn an instance of the in-parameters class.
	var spawned uintptr
	rr, _ := com.COMObject(inDef).Call(spawnSlot, 0, uintptr(unsafe.Pointer(&spawned)))
	fmt.Printf("SpawnInstance@%d -> 0x%08X obj=%#x valid=%v\n",
		spawnSlot, uint32(rr), spawned, com.COMObject(spawned).Valid())
	if rr != 0 || !com.COMObject(spawned).Valid() {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(spawned))

	// Set the input parameter.
	inName, _ := com.NewBSTR("gmInput")
	defer inName.Free()
	in := com.NewVariantUI4(0)
	rr, _ = com.COMObject(spawned).Call(4,
		uintptr(inName), 0, uintptr(unsafe.Pointer(&in)), 0)
	fmt.Printf("Put(gmInput=0) -> 0x%08X\n", uint32(rr))
	if rr != 0 {
		os.Exit(1)
	}

	paths := []string{
		fmt.Sprintf(`%s.InstanceName="ACPI\\PNP0C14\\APGe_0"`, className),
		fmt.Sprintf(`%s.InstanceName="ACPI\PNP0C14\APGe_0"`, className),
		className,
		fmt.Sprintf(`\\.\%s:%s.InstanceName="ACPI\\PNP0C14\\APGe_0"`, namespace, className),
		fmt.Sprintf(`\\localhost\%s\%s.InstanceName="ACPI\\PNP0C14\\APGe_0"`, namespace, className),
	}
	if pathVariant < 0 || pathVariant >= len(paths) {
		fmt.Printf("path variant out of range\n")
		os.Exit(2)
	}
	path, _ := com.NewBSTR(paths[pathVariant])
	defer path.Free()
	fmt.Printf("path=%s\n", paths[pathVariant])

	var outParams uintptr
	var flagsOut int32
	rr, _ = services.Call(20,
		uintptr(path),
		uintptr(method),
		0,
		0,
		uintptr(spawned),
		uintptr(unsafe.Pointer(&outParams)),
		uintptr(unsafe.Pointer(&flagsOut)),
	)
	fmt.Printf("ExecMethod -> 0x%08X out=%#x\n", uint32(rr), outParams)
	if rr == 0 && outParams != 0 && com.COMObject(outParams).Valid() {
		com.SetProxyBlanket(com.COMObject(outParams))
		readOut(outParams, outParam)
	}
}

// readOut reads a named output parameter.
func readOut(outParams uintptr, outParam string) {
	name, _ := com.NewBSTR(outParam)
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
	fmt.Printf("  Get(%s) -> 0x%08X vt=0x%04X value=%d (0x%X) err=%v\n",
		outParam, uint32(r), v.Type(), val, val, verr)
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
	if com.HHRESULT(r).Failed() {
		return 0, fmt.Errorf("HRESULT 0x%08X", uint32(r))
	}
	if services == 0 {
		return 0, fmt.Errorf("S_OK but no services pointer")
	}
	return com.COMObject(services), nil
}

// probe-marker
