//go:build windows

// Command wmifull drives a WMI method call with every remaining degree of
// freedom explored:
//
//   - the slot of IWbemClassObject::SpawnInstance
//   - using the in-parameters definition directly versus a spawned instance
//   - a range of plausible object paths
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

	namespace := "root\\WMI"
	className := "AcerGamingFunction"
	methodName := "GetGamingProfile"
	if len(os.Args) > 1 {
		namespace = os.Args[1]
	}
	if len(os.Args) > 2 {
		className = os.Args[2]
	}
	if len(os.Args) > 3 {
		methodName = os.Args[3]
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
	var class uintptr
	var scratch uintptr
	r, _ := services.Call(slotGetObject,
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
	r, _ = com.COMObject(class).Call(slotGetMethod,
		uintptr(method), 0,
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	fmt.Printf("GetMethod(%s) -> 0x%08X inDef=%#x outDef=%#x\n", methodName, uint32(r), inDef, outDef)
	if inDef == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(inDef))

	// Find SpawnInstance on the in-parameters definition.
	spawnSlot := -1
	for slot := 3; slot <= 24; slot++ {
		var obj uintptr
		rr, _ := com.COMObject(inDef).Call(slot, 0, uintptr(unsafe.Pointer(&obj)))
		if rr == 0 && obj != 0 {
			fmt.Printf("  spawninstance candidate slot %d -> 0x%08X obj=%#x\n", slot, uint32(rr), obj)
			if spawnSlot < 0 {
				spawnSlot = slot
				com.SetProxyBlanket(com.COMObject(obj))
			}
		}
	}
	fmt.Printf("spawn slot = %d\n", spawnSlot)

	// Build the input parameter value.
	inputName, _ := com.NewBSTR("gmInput")
	defer inputName.Free()
	in := com.NewVariantUI4(0)

	// Prepare two variants of the in-parameters object.
	putOnDef, _ := com.COMObject(inDef).Call(slotPut,
		uintptr(inputName), 0, uintptr(unsafe.Pointer(&in)), 0)
	fmt.Printf("Put on definition -> 0x%08X\n", uint32(putOnDef))

	var spawned uintptr
	if spawnSlot > 0 {
		rr, _ := com.COMObject(inDef).Call(spawnSlot, 0, uintptr(unsafe.Pointer(&spawned)))
		fmt.Printf("SpawnInstance -> 0x%08X obj=%#x\n", uint32(rr), spawned)
		if spawned != 0 {
			com.SetProxyBlanket(com.COMObject(spawned))
			rr, _ = com.COMObject(spawned).Call(slotPut,
				uintptr(inputName), 0, uintptr(unsafe.Pointer(&in)), 0)
			fmt.Printf("Put on spawned instance -> 0x%08X\n", uint32(rr))
		}
	}

	paths := []string{
		`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\0"`,
		`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"`,
		`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\1"`,
		`AcerGamingFunction`,
		`\\AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\0"`,
	}

	for _, useSpawned := range []bool{false, true} {
		inParams := inDef
		if useSpawned && spawned != 0 {
			inParams = spawned
		}
		label := "definition"
		if useSpawned {
			label = "spawned"
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
				uintptr(inParams),
				uintptr(unsafe.Pointer(&outParams)),
				uintptr(unsafe.Pointer(&flagsOut)),
			)
			fmt.Printf("ExecMethod[%s] %s -> 0x%08X out=%#x\n", label, p, uint32(rr), outParams)
			os.Stdout.Sync()
			bp.Free()
			if rr == 0 && outParams != 0 {
				com.SetProxyBlanket(com.COMObject(outParams))
				readOut(outParams)
				return
			}
		}
	}
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
