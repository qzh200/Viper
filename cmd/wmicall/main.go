//go:build windows

// Command wmicall performs a complete WMI method invocation end to end:
//
//	ConnectServer -> GetObject -> GetMethod -> Put -> ExecMethod -> Get
//
// It is the proof that the hand written COM layer can drive Acer's ACPI
// WMI provider.
package main

import (
	"fmt"
	"os"
	"unsafe"

	"viper/internal/com"
)

const (
	// Vtable slots, established empirically with cmd/slotfind.
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
	if len(os.Args) > 1 {
		namespace = os.Args[1]
	}
	className := "AcerGamingFunction"
	if len(os.Args) > 2 {
		className = os.Args[2]
	}
	methodName := "GetGamingProfile"
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
	fmt.Printf("connected to %s\n", namespace)

	// 1. Get the class object.
	classPath, _ := com.NewBSTR(className)
	defer classPath.Free()
	var class uintptr
	var scratch uintptr
	r, _ := services.Call(slotGetObject,
		uintptr(classPath),
		0,
		0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	fmt.Printf("GetObject(%s) -> HRESULT 0x%08X class=%#x\n", className, uint32(r), class)
	if com.HRESULT(r).Failed() || class == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(class))

	// 2. Get the method definition (in-parameters and out-parameters).
	method, _ := com.NewBSTR(methodName)
	defer method.Free()
	var inDef, outDef uintptr
	r, _ = com.COMObject(class).Call(slotGetMethod,
		uintptr(method), 0,
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	fmt.Printf("GetMethod(%s) -> HRESULT 0x%08X inDef=%#x outDef=%#x\n", methodName, uint32(r), inDef, outDef)
	if com.HRESULT(r).Failed() || inDef == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(inDef))

	// 3. Set the input parameter. The value is written straight onto the
	// in-parameters definition object, which avoids needing SpawnInstance.
	inputName, _ := com.NewBSTR("gmInput")
	defer inputName.Free()
	in := com.NewVariantUI4(0)
	r, _ = com.COMObject(inDef).Call(slotPut,
		uintptr(inputName), 0,
		uintptr(unsafe.Pointer(&in)),
		0,
	)
	fmt.Printf("Put(gmInput=0) -> HRESULT 0x%08X\n", uint32(r))

	// 4. Call the method on the ACPI instance.
	//
	// IWbemServices::ExecMethod takes seven parameters:
	//   strObjectPath, strMethodName, lFlags, ppCallResult,
	//   pInParams, ppOutParams, pFlags
	//
	// Putting the in-parameters object into the ppCallResult slot makes
	// WMI reject the call with CO_E_NOINTERFACE, so the order matters.
	for _, instancePath := range []string{
		`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\0"`,
		`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"`,
		`AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\1"`,
	} {
		path, _ := com.NewBSTR(instancePath)
		var outParams uintptr
		var flagsOut int32
		r, _ = services.Call(slotExecMethod,
			uintptr(path),                       // strObjectPath
			uintptr(method),                     // strMethodName
			0,                                   // lFlags
			0,                                   // ppCallResult
			uintptr(inDef),                      // pInParams
			uintptr(unsafe.Pointer(&outParams)), // ppOutParams
			uintptr(unsafe.Pointer(&flagsOut)),  // pFlags
		)
		fmt.Printf("ExecMethod(%s) -> HRESULT 0x%08X outParams=%#x\n", instancePath, uint32(r), outParams)
		path.Free()
		if com.HRESULT(r).Failed() || outParams == 0 {
			continue
		}
		com.SetProxyBlanket(com.COMObject(outParams))

		// 5. Read the output parameter.
		outName, _ := com.NewBSTR("gmOutput")
		defer outName.Free()
		var v com.VARIANT
		v.Init()
		var flavor int32
		var ctxOut uintptr
		r, _ = com.COMObject(outParams).Call(slotGet,
			uintptr(outName), 0,
			uintptr(unsafe.Pointer(&v)),
			uintptr(unsafe.Pointer(&flavor)),
			uintptr(unsafe.Pointer(&ctxOut)),
		)
		val, verr := v.AsUInt64()
		fmt.Printf("Get(gmOutput) -> HRESULT 0x%08X vt=0x%04X value=%d (0x%X) err=%v\n",
			uint32(r), v.Type(), val, val, verr)
		v.Clear()
		return
	}
	os.Exit(1)
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
