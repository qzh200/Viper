//go:build windows

// Command wmitest performs a WMI method call with fully configurable
// arguments and discovers the ExecMethod argument layout.
//
// Usage:
//
//	wmitest <namespace> <class> <method> <instancePath> <outParam>
//
// For example:
//
//	wmitest root\CIMV2 Win32_Process Terminate 'Win32_Process.Handle="999999"' ReturnValue
package main

import (
	"fmt"
	"os"
	"unsafe"

	"viper/internal/com"
)

func main() {
	if len(os.Args) < 6 {
		fmt.Println("usage: wmitest <ns> <class> <method> <instancePath> <outParam>")
		os.Exit(2)
	}
	namespace, className, methodName, instancePath, outParam :=
		os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]

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
	fmt.Printf("connected to %s\n", namespace)

	classPath, _ := com.NewBSTR(className)
	defer classPath.Free()
	var class, scratch uintptr
	r, _ := services.Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	fmt.Printf("GetObject(%s) -> 0x%08X class=%#x\n", className, uint32(r), class)
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
	if inDef != 0 {
		com.SetProxyBlanket(com.COMObject(inDef))
	}

	path, _ := com.NewBSTR(instancePath)
	defer path.Free()

	// Spawn an instance of the in-parameters class. The definition object
	// itself is rejected by the provider, so a real instance is required.
	type candidate struct {
		label string
		obj   uintptr
	}
	var candidates []candidate
	if inDef != 0 {
		// SpawnInstance lives at slot 34 on this build; the in-parameters
		// definition object itself is not accepted as pInParams.
		for _, spawnSlot := range []int{34, 13, 14, 15, 31, 22} {
			var spawned uintptr
			rr, _ := com.COMObject(inDef).Call(spawnSlot, 0, uintptr(unsafe.Pointer(&spawned)))
			if rr != 0 || !com.COMObject(spawned).Valid() {
				continue
			}
			com.SetProxyBlanket(com.COMObject(spawned))
			candidates = append(candidates, candidate{fmt.Sprintf("spawn@%d", spawnSlot), spawned})
		}
	}
	fmt.Printf("candidates: %d\n", len(candidates))

	for _, c := range candidates {
		// Set the input parameter on the candidate.
		if c.label != "definition" {
			inName, _ := com.NewBSTR("gmInput")
			in := com.NewVariantUI4(0)
			rr, _ := com.COMObject(c.obj).Call(4,
				uintptr(inName), 0, uintptr(unsafe.Pointer(&in)), 0)
			fmt.Printf("candidate %-12s Put -> 0x%08X\n", c.label, uint32(rr))
			inName.Free()
		}

		// The published ExecMethod signature, with a NULL
		// ppCallResult and a properly spawned in-parameters instance.
		// Only this shape is used: the six argument variant faults
		// inside the WMI provider.
		for _, slot := range []int{20, 28} {
			var outParams uintptr
			var flagsOut int32
			rr, _ := services.Call(slot,
				uintptr(path),
				uintptr(method),
				0,
				0,
				uintptr(c.obj),
				uintptr(unsafe.Pointer(&outParams)),
				uintptr(unsafe.Pointer(&flagsOut)),
			)
			fmt.Printf("  slot %d [%s] -> 0x%08X out=%#x\n", slot, c.label, uint32(rr), outParams)
			os.Stdout.Sync()
			if rr == 0 && outParams != 0 && com.COMObject(outParams).Valid() {
				com.SetProxyBlanket(com.COMObject(outParams))
				readOut(outParams, outParam)
				return
			}
		}
	}
	fmt.Println("no layout worked")
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
	if com.HRESULT(r).Failed() {
		return 0, fmt.Errorf("HRESULT 0x%08X", uint32(r))
	}
	if services == 0 {
		return 0, fmt.Errorf("S_OK but no services pointer")
	}
	return com.COMObject(services), nil
}
