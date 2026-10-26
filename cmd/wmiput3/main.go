//go:build windows

// Command wmiput3 sweeps the VARIANT type used for the input parameter.
//
// The Acer WMI provider is picky about the exact automation type, so the
// value is offered as UI4, UI8, I4 and BSTR, each with and without the
// UPDATE_ONLY flag.
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

	variants := []struct {
		name string
		make func() com.VARIANT
	}{
		{"VT_UI4", func() com.VARIANT { return com.NewVariantUI4(7) }},
		{"VT_UI8", func() com.VARIANT { return com.NewVariantUI8(7) }},
		{"VT_I4", func() com.VARIANT { return com.NewVariantI4(7) }},
		{"VT_BSTR", func() com.VARIANT {
			v, _ := com.NewVariantBSTR("7")
			return v
		}},
	}

	for _, v := range variants {
		for _, flags := range []uint32{0, 2} {
			var instance uintptr
			r, _ = com.COMObject(inDef).Call(15, 0, uintptr(unsafe.Pointer(&instance)))
			if r != 0 || instance == 0 {
				fmt.Printf("  spawn failed 0x%08X\n", uint32(r))
				continue
			}
			com.SetProxyBlanket(com.COMObject(instance))

			value := v.make()
			putRes, _ := com.COMObject(instance).Call(5,
				uintptr(inputName),
				uintptr(flags),
				uintptr(unsafe.Pointer(&value)),
				0,
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
			fmt.Printf("%-8s flags=%d -> Put 0x%08X  Get 0x%08X vt=0x%04X value=%d\n",
				v.name, flags, uint32(putRes), uint32(getRes), back.Type(), val)
			os.Stdout.Sync()
			back.Clear()
		}
	}
}
