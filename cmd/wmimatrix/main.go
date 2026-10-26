//go:build windows

// Command wmimatrix tries every combination of in-parameters construction
// and object path until the Acer WMI method answers.
package main

import (
	"fmt"
	"os"
	"unsafe"

	"viper/internal/com"
)

const (
	slotGetObject  = 6
	slotGetMethod  = 19
	slotGet        = 3
	slotPut        = 4
	slotExecMethod = 20
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
	r, _ := services.Call(slotGetObject,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	if class == 0 {
		fmt.Printf("GetObject failed 0x%08X\n", uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(class))

	methodName := "GetGamingProfile"
	if len(os.Args) > 1 {
		methodName = os.Args[1]
	}
	method, _ := com.NewBSTR(methodName)
	defer method.Free()
	var inDef, outDef uintptr
	r, _ = com.COMObject(class).Call(slotGetMethod,
		uintptr(method), 0,
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	if inDef == 0 {
		fmt.Println("GetMethod returned no in-parameters (method takes none)")
	} else {
		com.SetProxyBlanket(com.COMObject(inDef))
	}

	// Read the declared CIM type of the input parameter so the variant
	// type matches what the provider expects.
	inputName, _ := com.NewBSTR("gmInput")
	defer inputName.Free()
	reportCimType(inDef)

	paths := []string{
		`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"`,
		`AcerGamingFunction.InstanceName="ACPI\PNP0C14\APGe_0"`,
		`AcerGamingFunction`,
		`\\AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"`,
		`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\0"`,
	}

	// Candidate in-parameters objects: the definition itself, plus a
	// spawned instance from each candidate SpawnInstance slot.
	type candidate struct {
		label string
		obj   uintptr
	}
	var candidates []candidate
	if inDef != 0 {
		candidates = append(candidates, candidate{"definition", inDef})
	} else {
		// A method with no input parameters is invoked with a NULL
		// in-parameters object.
		candidates = append(candidates, candidate{"none", 0})
	}

	for _, spawnSlot := range []int{13, 14, 15} {
		if inDef == 0 {
			break
		}
		var obj uintptr
		rr, _ := com.COMObject(inDef).Call(spawnSlot, 0, uintptr(unsafe.Pointer(&obj)))
		if rr != 0 || !com.COMObject(obj).Valid() {
			continue
		}
		com.SetProxyBlanket(com.COMObject(obj))
		candidates = append(candidates, candidate{fmt.Sprintf("spawn@%d", spawnSlot), obj})
	}

	for _, c := range candidates {
		// Put the input value onto this candidate.
		if c.obj != 0 {
			in := com.NewVariantUI4(0)
			putRes, _ := com.COMObject(c.obj).Call(slotPut,
				uintptr(inputName), 0, uintptr(unsafe.Pointer(&in)), 0)
			fmt.Printf("candidate %-14s Put -> 0x%08X\n", c.label, uint32(putRes))
			os.Stdout.Sync()
		}

		for _, p := range paths {
			bp, _ := com.NewBSTR(p)
			var outParams uintptr
			var flagsOut int32
			rr, _ := services.Call(slotExecMethod,
				uintptr(bp),
				uintptr(method),
				0,
				0,
				uintptr(c.obj),
				uintptr(unsafe.Pointer(&outParams)),
				uintptr(unsafe.Pointer(&flagsOut)),
			)
			fmt.Printf("  ExecMethod[%s] %s -> 0x%08X out=%#x\n", c.label, p, uint32(rr), outParams)
			os.Stdout.Sync()
			bp.Free()
			if rr == 0 && outParams != 0 {
				com.SetProxyBlanket(com.COMObject(outParams))
				readOut(outParams)
				return
			}
		}
	}
	fmt.Println("no combination worked")
}

// reportCimType prints the declared CIM type of an input parameter.
func reportCimType(inDef uintptr) {
	if inDef == 0 {
		return
	}
	name, _ := com.NewBSTR("gmInput")
	defer name.Free()
	var v com.VARIANT
	v.Init()
	var flavor int32
	var ctxOut uintptr
	r, _ := com.COMObject(inDef).Call(slotGet,
		uintptr(name), 0,
		uintptr(unsafe.Pointer(&v)),
		uintptr(unsafe.Pointer(&flavor)),
		uintptr(unsafe.Pointer(&ctxOut)),
	)
	fmt.Printf("in-params gmInput -> 0x%08X vt=0x%04X flavor=%d\n", uint32(r), v.Type(), flavor)
	v.Clear()
}

// readOut reads the gmOutput parameter from the out-parameters object.
func readOut(outParams uintptr) {
	outName, _ := com.NewBSTR("gmOutput")
	defer outName.Free()
	var v com.VARIANT
	v.Init()
	var flavor int32
	var ctxOut uintptr
	r, _ := com.COMObject(outParams).Call(slotGet,
		uintptr(outName), 0,
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
