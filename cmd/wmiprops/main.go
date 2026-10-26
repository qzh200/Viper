//go:build windows

// Command wmiprops lists the properties of the WMI objects involved in a
// method call: the class, the in-parameters definition and the
// out-parameters definition.
//
// Usage: wmiprops <ns> <class> <method>
package main

import (
	"fmt"
	"os"
	"unsafe"

	"viper/internal/com"
)

// IWbemClassObject::GetNames is slot 34 on this build.
const slotGetNames = 34

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: wmiprops <ns> <class> <method>")
		os.Exit(2)
	}
	namespace, className, methodName := os.Args[1], os.Args[2], os.Args[3]

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

	classPath, _ := com.NewBSTR(className)
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
	dumpNames(class, "class")

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
		dumpNames(inDef, "in-params definition")
	}
	if outDef != 0 {
		com.SetProxyBlanket(com.COMObject(outDef))
		dumpNames(outDef, "out-params definition")
	}
}

// dumpNames prints the property names of a class object.
func dumpNames(obj uintptr, label string) {
	var names, quals uintptr
	r, _ := com.COMObject(obj).Call(slotGetNames,
		0,
		uintptr(unsafe.Pointer(&names)),
		uintptr(unsafe.Pointer(&quals)),
	)
	fmt.Printf("%-24s GetNames -> 0x%08X names=%#x\n", label, uint32(r), names)
	if names != 0 {
		list, err := com.SafeArrayBSTRs(names)
		fmt.Printf("%-24s   %v %v\n", label, err, list)
	}
	if quals != 0 {
		com.COMObject(quals).Release()
	}
}
