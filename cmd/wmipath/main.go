//go:build windows

// Command wmipath finds the real instance path of a WMI class.
//
// It sweeps the IWbemServices vtable for ExecQuery, enumerates the class
// instances and prints their InstanceName and __RELPATH.
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

	namespace := "root\\WMI"
	className := "AcerGamingFunction"
	if len(os.Args) > 1 {
		namespace = os.Args[1]
	}
	if len(os.Args) > 2 {
		className = os.Args[2]
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

	lang, _ := com.NewBSTR("WQL")
	query, _ := com.NewBSTR("SELECT * FROM " + className)
	defer lang.Free()
	defer query.Free()

	for slot := 4; slot <= 8; slot++ {
		var enum uintptr
		var flagsOut int32
		// Six argument shape: strQueryLanguage, strQuery, lFlags,
		// ppCallResult, ppEnum, pFlags.
		r, _ := services.Call(slot,
			uintptr(lang),
			uintptr(query),
			0x30,
			0,
			uintptr(unsafe.Pointer(&enum)),
			uintptr(unsafe.Pointer(&flagsOut)),
		)
		fmt.Printf("slot %d -> HRESULT 0x%08X enum=%#x\n", slot, uint32(r), enum)
		os.Stdout.Sync()
		if com.HRESULT(r).Failed() || enum == 0 {
			continue
		}
		com.SetProxyBlanket(com.COMObject(enum))
		dumpInstances(enum)
		return
	}
}

// dumpInstances pulls every instance out of an enumerator and prints the
// interesting properties.
func dumpInstances(enum uintptr) {
	timeout := int64(-1)
	for i := 0; i < 8; i++ {
		var obj uintptr
		var returned uint32
		r, _ := com.COMObject(enum).Call(3,
			uintptr(uint64(timeout)),
			1,
			uintptr(unsafe.Pointer(&obj)),
			uintptr(unsafe.Pointer(&returned)),
		)
		fmt.Printf("  Next -> HRESULT 0x%08X obj=%#x returned=%d\n", uint32(r), obj, returned)
		os.Stdout.Sync()
		if obj == 0 || returned == 0 {
			return
		}
		com.SetProxyBlanket(com.COMObject(obj))
		for _, prop := range []string{"__RELPATH", "InstanceName", "Active"} {
			name, _ := com.NewBSTR(prop)
			var v com.VARIANT
			v.Init()
			var flavor int32
			var ctxOut uintptr
			rr, _ := com.COMObject(obj).Call(3,
				uintptr(name), 0,
				uintptr(unsafe.Pointer(&v)),
				uintptr(unsafe.Pointer(&flavor)),
				uintptr(unsafe.Pointer(&ctxOut)),
			)
			val, verr := v.AsString()
			fmt.Printf("    %s -> HRESULT 0x%08X vt=0x%04X value=%q err=%v\n",
				prop, uint32(rr), v.Type(), val, verr)
			os.Stdout.Sync()
			v.Clear()
			name.Free()
		}
	}
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
