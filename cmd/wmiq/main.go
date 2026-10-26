//go:build windows

// Command wmiq runs IWbemServices::ExecQuery with the six argument shape
// and prints the resulting instances.
//
// Usage: wmiq <ns> <query>
package main

import (
	"fmt"
	"os"
	"unsafe"

	"viper/internal/com"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: wmiq <ns> <query>")
		os.Exit(2)
	}
	namespace, queryText := os.Args[1], os.Args[2]

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

	lang, _ := com.NewBSTR("WQL")
	query, _ := com.NewBSTR(queryText)
	defer lang.Free()
	defer query.Free()

	// IWbemServices::ExecQuery (slot 20) with six arguments:
	//   strQueryLanguage, strQuery, lFlags, ppCallResult, ppEnum, pFlags
	var enum uintptr
	var flagsOut int32
	r, _ = com.COMObject(services).Call(20,
		uintptr(lang),
		uintptr(query),
		0x30, // WBEM_FLAG_RETURN_IMMEDIATELY | WBEM_FLAG_FORWARD_ONLY
		0,    // ppCallResult
		uintptr(unsafe.Pointer(&enum)),
		uintptr(unsafe.Pointer(&flagsOut)),
	)
	fmt.Printf("ExecQuery -> 0x%08X enum=%#x\n", uint32(r), enum)
	if int32(r) < 0 || enum == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(enum))

	// IEnumWbemClassObject::Next (slot 4).
	timeout := int64(-1)
	for i := 0; i < 4; i++ {
		var obj uintptr
		var returned uint32
		r, _ = com.COMObject(enum).Call(4,
			uintptr(uint64(timeout)),
			1,
			uintptr(unsafe.Pointer(&obj)),
			uintptr(unsafe.Pointer(&returned)),
		)
		fmt.Printf("Next -> 0x%08X obj=%#x returned=%d\n", uint32(r), obj, returned)
		if obj == 0 || returned == 0 {
			return
		}
		com.SetProxyBlanket(com.COMObject(obj))

		for _, prop := range []string{"__RELPATH", "InstanceName", "Active"} {
			name, _ := com.NewBSTR(prop)
			var v com.VARIANT
			v.Init()
			var flavor int32
			var ctx uintptr
			rr, _ := com.COMObject(obj).Call(4,
				uintptr(name), 0,
				uintptr(unsafe.Pointer(&v)),
				uintptr(unsafe.Pointer(&flavor)),
				uintptr(unsafe.Pointer(&ctx)),
			)
			val, verr := v.AsString()
			fmt.Printf("  %s -> 0x%08X vt=0x%04X value=%q err=%v\n",
				prop, uint32(rr), v.Type(), val, verr)
			v.Clear()
			name.Free()
		}
	}
}
