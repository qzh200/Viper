//go:build windows

// Command slotfind probes one WMI vtable slot per process.
//
// Probing unknown slots can fault the process, so every probe runs in its
// own invocation and the driver script tolerates a non-zero exit.
//
// Usage:
//
//	slotfind connect
//	slotfind class-get <slot>
//	slotfind class-put <slot>
//	slotfind class-getmethod <slot>
//	slotfind class-spawninstance <slot>
//	slotfind class-getnames <slot>
//	slotfind services-execmethod <slot>
package main

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"viper/internal/com"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("missing mode")
		os.Exit(2)
	}
	mode := os.Args[1]

	if err := com.InitCOM(); err != nil {
		fmt.Println("InitCOM:", err)
		os.Exit(1)
	}
	defer com.UninitCOM()
	if err := com.InitCOMSecurity(); err != nil {
		fmt.Println("InitCOMSecurity:", err)
	}

	switch mode {
	case "connect":
		svc := mustConnect()
		fmt.Printf("CONNECTED ptr=%#x\n", uintptr(svc))
	case "class-get":
		classProbe("get", slotArg())
	case "class-put":
		classProbe("put", slotArg())
	case "class-getmethod":
		classProbe("getmethod", slotArg())
	case "class-spawninstance":
		classProbe("spawninstance", slotArg())
	case "class-getnames":
		classProbe("getnames", slotArg())
	case "services-execmethod":
		execMethodProbe(slotArg())
	case "inparams-spawn":
		inParamsSpawnProbe(slotArg())
	case "real-execmethod":
		realExecMethodProbe(slotArg())
	default:
		fmt.Println("unknown mode", mode)
		os.Exit(2)
	}
}

// slotArg parses the slot number from the command line.
func slotArg() int {
	if len(os.Args) < 3 {
		fmt.Println("missing slot")
		os.Exit(2)
	}
	n, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("bad slot:", err)
		os.Exit(2)
	}
	return n
}

// report prints a probe result and flushes it, so that a fault in a later
// step does not swallow the output.
func report(format string, args ...any) {
	fmt.Printf(format, args...)
	os.Stdout.Sync()
}

// connect calls IWbemLocator::ConnectServer.
//
// The eight argument layout below was determined empirically with
// cmd/validate: the namespace BSTR is the first argument and the
// IWbemServices out-pointer is the last. The nine argument layout that
// appears in most published headers fails with WBEM_E_NOT_SUPPORTED on
// this build of wbemprov.
func connect(locator com.COMObject, ns string) (com.COMObject, error) {
	b, err := com.NewBSTR(ns)
	if err != nil {
		return 0, err
	}
	defer b.Free()

	var services uintptr
	r, err := locator.Call(3,
		uintptr(b),                         // strNamespacePath
		0,                                  // strServer
		0,                                  // strLocale
		0,                                  // strUser
		0,                                  // strPassword
		0,                                  // lSecurityFlags
		0,                                  // strAuthority
		uintptr(unsafe.Pointer(&services)), // out: IWbemServices
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

// mustConnect opens the root\CIMV2 namespace or exits.
func mustConnect() com.COMObject {
	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		fmt.Println("CoCreateInstance:", err)
		os.Exit(1)
	}
	defer locator.Release()

	svc, err := connect(locator, "root\\CIMV2")
	if err != nil {
		fmt.Println("ConnectServer:", err)
		os.Exit(1)
	}
	com.SetProxyBlanket(svc)
	return svc
}

// mustClass returns the class object named on the command line, or exits.
//
// Usage: slotfind <mode> <slot> [namespace] [class] [method]
func mustClass() com.COMObject {
	namespace := "root\\CIMV2"
	className := "Win32_OperatingSystem"
	if len(os.Args) > 3 {
		namespace = os.Args[3]
	}
	if len(os.Args) > 4 {
		className = os.Args[4]
	}

	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		fmt.Println("CoCreateInstance:", err)
		os.Exit(1)
	}
	defer locator.Release()

	svc, err := connect(locator, namespace)
	if err != nil {
		fmt.Println("ConnectServer:", err)
		os.Exit(1)
	}
	com.SetProxyBlanket(svc)

	path, _ := com.NewBSTR(className)
	defer path.Free()

	var obj uintptr
	var ctxOut uintptr
	r, _ := svc.Call(6, // GetObject
		uintptr(path),
		0,
		0,
		uintptr(unsafe.Pointer(&obj)),
		uintptr(unsafe.Pointer(&ctxOut)),
	)
	if com.HRESULT(r).Failed() || obj == 0 {
		report("GetObject(%s) failed 0x%08X\n", className, uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(obj))
	return com.COMObject(obj)
}

// classProbe calls one candidate IWbemClassObject slot.
//
// Every out-parameter slot receives a real writable buffer, because a NULL
// in a slot the callee writes to faults the process.
func classProbe(kind string, slot int) {
	class := mustClass()

	switch kind {
	case "get":
		prop := "ClassName"
		if len(os.Args) > 5 {
			prop = os.Args[5]
		}
		name, _ := com.NewBSTR(prop)
		defer name.Free()
		var v com.VARIANT
		v.Init()
		var flavor int32
		var ctxOut uintptr
		r, _ := class.Call(slot,
			uintptr(name), 0,
			uintptr(unsafe.Pointer(&v)),
			uintptr(unsafe.Pointer(&flavor)),
			uintptr(unsafe.Pointer(&ctxOut)),
		)
		val, verr := v.AsString()
		report("SLOT %d class-get(%s) -> HRESULT 0x%08X vt=0x%04X value=%q err=%v\n",
			slot, prop, uint32(r), v.Type(), val, verr)
		v.Clear()
	case "put":
		name, _ := com.NewBSTR("NoSuchProperty")
		defer name.Free()
		v := com.NewVariantI4(1)
		r, _ := class.Call(slot,
			uintptr(name), 0,
			uintptr(unsafe.Pointer(&v)),
			0,
		)
		report("SLOT %d class-put -> HRESULT 0x%08X\n", slot, uint32(r))
	case "getmethod":
		methodName := "NoSuchMethod"
		if len(os.Args) > 5 {
			methodName = os.Args[5]
		}
		name, _ := com.NewBSTR(methodName)
		defer name.Free()
		var inDef, outDef uintptr
		r, _ := class.Call(slot,
			uintptr(name), 0,
			uintptr(unsafe.Pointer(&inDef)),
			uintptr(unsafe.Pointer(&outDef)),
		)
		report("SLOT %d class-getmethod(%s) -> HRESULT 0x%08X inDef=%#x outDef=%#x\n",
			slot, methodName, uint32(r), inDef, outDef)
	case "spawninstance":
		var obj uintptr
		r, _ := class.Call(slot, 0, uintptr(unsafe.Pointer(&obj)))
		report("SLOT %d class-spawninstance -> HRESULT 0x%08X obj=%v\n", slot, uint32(r), obj != 0)
	case "getnames":
		var names, quals uintptr
		r, _ := class.Call(slot, 0, uintptr(unsafe.Pointer(&names)), uintptr(unsafe.Pointer(&quals)))
		report("SLOT %d class-getnames -> HRESULT 0x%08X names=%v\n", slot, uint32(r), names != 0)
	}
}

// inParamsSpawnProbe calls one candidate slot on the in-parameters
// definition object of a real method, which is where SpawnInstance lives.
//
// Usage: slotfind inparams-spawn <slot> [namespace] [class] [method]
func inParamsSpawnProbe(slot int) {
	namespace := "root\\WMI"
	className := "AcerGamingFunction"
	methodName := "GetGamingProfile"
	if len(os.Args) > 3 {
		namespace = os.Args[3]
	}
	if len(os.Args) > 4 {
		className = os.Args[4]
	}
	if len(os.Args) > 5 {
		methodName = os.Args[5]
	}

	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		fmt.Println("CoCreateInstance:", err)
		os.Exit(1)
	}
	defer locator.Release()

	svc, err := connect(locator, namespace)
	if err != nil {
		fmt.Println("ConnectServer:", err)
		os.Exit(1)
	}
	com.SetProxyBlanket(svc)

	classPath, _ := com.NewBSTR(className)
	defer classPath.Free()
	var class, scratch uintptr
	r, _ := svc.Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	if class == 0 {
		report("GetObject failed 0x%08X\n", uint32(r))
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
		report("GetMethod failed 0x%08X\n", uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(inDef))

	var obj uintptr
	rr, _ := com.COMObject(inDef).Call(slot, 0, uintptr(unsafe.Pointer(&obj)))
	report("SLOT %d inparams-spawn -> HRESULT 0x%08X obj=%#x\n", slot, uint32(rr), obj)
}

// realExecMethodProbe drives the whole chain with a real method and then
// calls one candidate IWbemServices slot with the ExecMethod shape.
//
// Usage: slotfind real-execmethod <slot> [namespace] [class] [method]
func realExecMethodProbe(slot int) {
	namespace := "root\\WMI"
	className := "AcerGamingFunction"
	methodName := "GetGamingProfile"
	if len(os.Args) > 3 {
		namespace = os.Args[3]
	}
	if len(os.Args) > 4 {
		className = os.Args[4]
	}
	if len(os.Args) > 5 {
		methodName = os.Args[5]
	}

	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		fmt.Println("CoCreateInstance:", err)
		os.Exit(1)
	}
	defer locator.Release()

	svc, err := connect(locator, namespace)
	if err != nil {
		fmt.Println("ConnectServer:", err)
		os.Exit(1)
	}
	com.SetProxyBlanket(svc)

	classPath, _ := com.NewBSTR(className)
	defer classPath.Free()
	var class, scratch uintptr
	r, _ := svc.Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	if class == 0 {
		report("GetObject failed 0x%08X\n", uint32(r))
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
		report("GetMethod failed 0x%08X\n", uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(inDef))

	inputName, _ := com.NewBSTR("gmInput")
	defer inputName.Free()
	in := com.NewVariantUI4(0)
	rr, _ := com.COMObject(inDef).Call(4,
		uintptr(inputName), 0, uintptr(unsafe.Pointer(&in)), 0)
	report("Put -> 0x%08X\n", uint32(rr))

	instancePath, _ := com.NewBSTR(`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"`)
	defer instancePath.Free()

	// Several plausible ExecMethod signatures are tried, because a
	// mismatched layout silently writes the out-parameter into the wrong
	// slot instead of failing.
	layouts := []struct {
		name string
		args func(out *uintptr, flags *int32) []uintptr
	}{
		{"7:path,method,flags,callResult,in,out,pFlags", func(out *uintptr, flags *int32) []uintptr {
			return []uintptr{uintptr(instancePath), uintptr(method), 0, 0, uintptr(inDef),
				uintptr(unsafe.Pointer(out)), uintptr(unsafe.Pointer(flags))}
		}},
		{"6:path,method,flags,in,out,pFlags", func(out *uintptr, flags *int32) []uintptr {
			return []uintptr{uintptr(instancePath), uintptr(method), 0, uintptr(inDef),
				uintptr(unsafe.Pointer(out)), uintptr(unsafe.Pointer(flags))}
		}},
		{"5:path,method,flags,in,out", func(out *uintptr, flags *int32) []uintptr {
			return []uintptr{uintptr(instancePath), uintptr(method), 0, uintptr(inDef),
				uintptr(unsafe.Pointer(out))}
		}},
		{"8:path,method,flags,callResult,in,out,pFlags,ctx", func(out *uintptr, flags *int32) []uintptr {
			return []uintptr{uintptr(instancePath), uintptr(method), 0, 0, uintptr(inDef),
				uintptr(unsafe.Pointer(out)), uintptr(unsafe.Pointer(flags)), 0}
		}},
	}

	for _, l := range layouts {
		var outParams uintptr
		var flagsOut int32
		rr, _ := svc.Call(slot, l.args(&outParams, &flagsOut)...)
		report("SLOT %d layout %s -> HRESULT 0x%08X out=%#x\n", slot, l.name, uint32(rr), outParams)
		if rr == 0 && outParams != 0 && com.COMObject(outParams).Valid() {
			com.SetProxyBlanket(com.COMObject(outParams))
			outName, _ := com.NewBSTR("gmOutput")
			var v com.VARIANT
			v.Init()
			var flavor int32
			var ctxOut uintptr
			r2, _ := com.COMObject(outParams).Call(3,
				uintptr(outName), 0,
				uintptr(unsafe.Pointer(&v)),
				uintptr(unsafe.Pointer(&flavor)),
				uintptr(unsafe.Pointer(&ctxOut)),
			)
			val, verr := v.AsUInt64()
			report("  Get(gmOutput) -> 0x%08X vt=0x%04X value=%d (0x%X) err=%v\n",
				uint32(r2), v.Type(), val, val, verr)
			v.Clear()
			return
		}
	}
}

// execMethodProbe calls one candidate IWbemServices slot with the
// ExecMethod shape.
func execMethodProbe(slot int) {
	services := mustConnect()

	objPath, _ := com.NewBSTR("StdRegProv")
	method, _ := com.NewBSTR("NoSuchMethod")
	defer objPath.Free()
	defer method.Free()

	var out uintptr
	var flagsOut int32
	r, _ := services.Call(slot,
		uintptr(objPath), uintptr(method),
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
		uintptr(unsafe.Pointer(&flagsOut)),
	)
	report("SLOT %d services-execmethod -> HRESULT 0x%08X out=%v\n", slot, uint32(r), out != 0)
}
