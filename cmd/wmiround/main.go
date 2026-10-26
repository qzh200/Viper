//go:build windows

// Command wmiround resolves the instance path with ExecQuery and then
// invokes the method on exactly that path, in one process.
//
// This removes any guesswork about how the object path has to be escaped.
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

	// Resolve the instance path with ExecQuery.
	path := resolvePath(com.COMObject(services), "AcerGamingFunction")
	if path == "" {
		fmt.Println("could not resolve the instance path")
		os.Exit(1)
	}
	fmt.Printf("resolved path: %s\n", path)

	// Invoke the method on it.
	classPath, _ := com.NewBSTR("AcerGamingFunction")
	defer classPath.Free()
	var class uintptr
	r, _ = com.COMObject(services).Call(6,
		uintptr(classPath), 0, 0,
		uintptr(unsafe.Pointer(&class)),
		0,
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

	var instance uintptr
	r, _ = com.COMObject(inDef).Call(15, 0, uintptr(unsafe.Pointer(&instance)), 0)
	fmt.Printf("SpawnInstance -> 0x%08X instance=%#x\n", uint32(r), instance)
	if r != 0 || instance == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(instance))

	inputName, _ := com.NewBSTR("gmInput")
	defer inputName.Free()
	value := com.NewVariantI4(0)
	r, _ = com.COMObject(instance).Call(5,
		uintptr(inputName), 0, uintptr(unsafe.Pointer(&value)), 0)
	fmt.Printf("Put(gmInput=0) -> 0x%08X\n", uint32(r))

	objectPath, _ := com.NewBSTR(path)
	defer objectPath.Free()
	var outParams uintptr
	r, _ = com.COMObject(services).Call(24,
		uintptr(objectPath),
		uintptr(method),
		0,
		0,
		uintptr(instance),
		uintptr(unsafe.Pointer(&outParams)),
		0,
	)
	fmt.Printf("ExecMethod -> 0x%08X outParams=%#x\n", uint32(r), outParams)
	if int32(r) < 0 || outParams == 0 {
		os.Exit(1)
	}
	com.SetProxyBlanket(com.COMObject(outParams))

	outName, _ := com.NewBSTR("gmOutput")
	defer outName.Free()
	var out com.VARIANT
	out.Init()
	var flavor int32
	r, _ = com.COMObject(outParams).Call(4,
		uintptr(outName), 0,
		uintptr(unsafe.Pointer(&out)),
		uintptr(unsafe.Pointer(&flavor)),
		0,
	)
	val, verr := out.AsUInt64()
	fmt.Printf("Get(gmOutput) -> 0x%08X vt=0x%04X value=%d (0x%X) err=%v\n",
		uint32(r), out.Type(), val, val, verr)
	out.Clear()
}

// resolvePath queries the class and returns the __RELPATH of its instance.
func resolvePath(services com.COMObject, className string) string {
	lang, _ := com.NewBSTR("WQL")
	query, _ := com.NewBSTR("SELECT * FROM " + className)
	defer lang.Free()
	defer query.Free()

	var enum uintptr
	r, _ := services.Call(20,
		uintptr(lang), uintptr(query), 0x30, 0,
		uintptr(unsafe.Pointer(&enum)),
		0,
	)
	if com.HRESULT(r).Failed() || enum == 0 {
		fmt.Printf("ExecQuery failed 0x%08X\n", uint32(r))
		return ""
	}
	com.SetProxyBlanket(com.COMObject(enum))

	timeout := int64(-1)
	var obj uintptr
	var returned uint32
	r, _ = com.COMObject(enum).Call(4,
		uintptr(uint64(timeout)), 1,
		uintptr(unsafe.Pointer(&obj)),
		uintptr(unsafe.Pointer(&returned)),
	)
	if obj == 0 || returned == 0 {
		fmt.Printf("Next failed 0x%08X\n", uint32(r))
		return ""
	}
	com.SetProxyBlanket(com.COMObject(obj))

	name, _ := com.NewBSTR("__RELPATH")
	defer name.Free()
	var v com.VARIANT
	v.Init()
	var flavor int32
	r, _ = com.COMObject(obj).Call(4,
		uintptr(name), 0,
		uintptr(unsafe.Pointer(&v)),
		uintptr(unsafe.Pointer(&flavor)),
		0,
	)
	if com.HRESULT(r).Failed() {
		fmt.Printf("Get(__RELPATH) failed 0x%08X\n", uint32(r))
		return ""
	}
	path, err := v.AsString()
	if err != nil {
		fmt.Println("__RELPATH:", err)
		return ""
	}
	return path
}
