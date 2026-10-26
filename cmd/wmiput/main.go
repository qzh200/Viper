//go:build windows

// Command wmiput finds the Put argument combination that actually stores
// an input parameter value.
//
// IWbemClassObject::Put(BSTR name, long flags, VARIANT *value, int type)
// has a "type" argument that names the CIM type of the property. Passing
// CIM_ILLEGAL (0) is what a naive implementation does and it is rejected
// by the WMI provider.
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

	ns, _ := com.NewBSTR("root\\WMI")
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

	classPath, _ := com.NewBSTR("AcerGamingFunction")
	defer classPath.Free()
	var class, scratch uintptr
	r, _ = com.COMObject(services).Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		uintptr(unsafe.Pointer(&scratch)),
	)
	if class == 0 {
		fmt.Printf("GetObject failed 0x%08X\n", uint32(r))
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(class))

	method, _ := com.NewBSTR("GetGamingProfile")
	defer method.Free()
	var inDef, outDef uintptr
	r, _ = com.COMObject(class).Call(19,
		uintptr(method), 0,
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	fmt.Printf("GetMethod -> 0x%08X inDef=%#x\n", uint32(r), inDef)
	if inDef == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(inDef))

	inputName, _ := com.NewBSTR("gmInput")
	defer inputName.Free()

	// WMI property types worth trying for a UInt32 parameter.
	types := []struct {
		name string
		id   uint32
	}{
		{"CIM_ILLEGAL", 0},
		{"CIM_UINT32", 19},
		{"CIM_UINT64", 21},
		{"CIM_SINT32", 3},
		{"CIM_BSTR", 8},
		{"CIM_UINT16", 18},
		{"CIM_SINT64", 20},
	}
	flags := []struct {
		name string
		id   uint32
	}{
		{"0", 0},
		{"WBEM_FLAG_UPDATE_ONLY(2)", 2},
	}

	for _, ty := range types {
		for _, fl := range flags {
			// A fresh instance for every attempt.
			var instance uintptr
			r, _ = com.COMObject(inDef).Call(15, 0, uintptr(unsafe.Pointer(&instance)))
			if r != 0 || instance == 0 {
				fmt.Printf("  spawn failed 0x%08X\n", uint32(r))
				continue
			}
			com.SetProxyBlanket(com.COMObject(instance))

			input := com.NewVariantUI4(7)
			putRes, _ := com.COMObject(instance).Call(5,
				uintptr(inputName),
				uintptr(fl.id),
				uintptr(unsafe.Pointer(&input)),
				uintptr(ty.id),
			)

			var back com.VARIANT
			back.Init()
			var flavor int32
			var ctx uintptr
			getRes, _ := com.COMObject(instance).Call(4,
				uintptr(inputName), 0,
				uintptr(unsafe.Pointer(&back)),
				uintptr(unsafe.Pointer(&flavor)),
				uintptr(unsafe.Pointer(&ctx)),
			)
			val, _ := back.AsUInt64()
			fmt.Printf("flags=%-20s type=%-12s -> Put 0x%08X  Get 0x%08X vt=0x%04X value=%d\n",
				fl.name, ty.name, uint32(putRes), uint32(getRes), back.Type(), val)
			os.Stdout.Sync()
			back.Clear()
		}
	}
}
