//go:build windows

// Command wmidiag isolates where the WMI connection fails.
//
// It walks the chain step by step and reports the HRESULT of each stage:
//
//	CoInitializeSecurity -> CoCreateInstance -> IUnknown refcounting ->
//	ConnectServer (several argument shapes)
package main

import (
	"fmt"
	"unsafe"

	"viper/internal/com"
)

func main() {
	// The order matters: CoInitializeEx must run before
	// CoInitializeSecurity, otherwise the latter fails with
	// CO_E_NOTINITIALIZED (0x800401F0).
	if err := com.InitCOM(); err != nil {
		fmt.Println("InitCOM:", err)
		return
	}
	defer com.UninitCOM()

	// COM security must be established before any COM object is created,
	// otherwise it fails with RPC_E_TOO_LATE.
	if err := com.InitCOMSecurity(); err != nil {
		fmt.Println("InitCOMSecurity:", err)
	} else {
		fmt.Println("InitCOMSecurity: ok")
	}

	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		fmt.Println("CoCreateInstance:", err)
		return
	}
	defer locator.Release()
	fmt.Printf("CoCreateInstance: ok, ptr=%#x\n", uintptr(locator))

	// Prove that the vtable dispatch works at all: AddRef must return a
	// refcount of at least 2 for a freshly created object.
	addRef, err := locator.Call(1)
	if err != nil {
		fmt.Println("AddRef failed:", err)
		return
	}
	fmt.Printf("IUnknown::AddRef -> refcount=%d\n", addRef)
	rel, _ := locator.Call(2)
	fmt.Printf("IUnknown::Release -> refcount=%d\n", rel)

	// Now try ConnectServer with a few argument shapes.
	for _, shape := range []struct {
		name string
		ns   string
	}{
		{"server=NULL ns=root\\CIMV2", "root\\CIMV2"},
		{"server=NULL ns=root\\WMI", "root\\WMI"},
		{"server=NULL ns=root\\cimv2", "root\\cimv2"},
	} {
		svc, err := connect(locator, shape.ns)
		if err != nil {
			fmt.Printf("ConnectServer(%s) -> %v\n", shape.name, err)
			continue
		}
		fmt.Printf("ConnectServer(%s) -> OK ptr=%#x\n", shape.name, uintptr(svc))
		com.SetProxyBlanket(svc)
		probeServices(svc)
		return
	}
}

// connect calls IWbemLocator::ConnectServer with the canonical nine
// argument shape.
func connect(locator com.COMObject, ns string) (com.COMObject, error) {
	b, err := com.NewBSTR(ns)
	if err != nil {
		return 0, err
	}
	defer b.Free()

	var services uintptr
	r, err := locator.Call(3,
		0,                                  // strServer
		uintptr(b),                         // strNamespacePath
		0,                                  // strLocale
		0,                                  // strUser
		0,                                  // strPassword
		0,                                  // lSecurityFlags
		0,                                  // strAuthority
		0,                                  // ppCtx
		uintptr(unsafe.Pointer(&services)), // ppLoc
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

// probeServices scans the IWbemServices vtable for the three methods we
// need, matching distinctive WMI error codes.
func probeServices(services com.COMObject) {
	fmt.Println("  -- ExecQuery candidates (expect WBEM_E_INVALID_QUERY 0x80041017) --")
	for slot := 3; slot <= 26; slot++ {
		lang, _ := com.NewBSTR("WQL")
		query, _ := com.NewBSTR("SELECT FROM")
		var enum uintptr
		r, _ := services.Call(slot,
			uintptr(lang), uintptr(query),
			0x30,
			uintptr(unsafe.Pointer(&enum)),
			0,
		)
		fmt.Printf("    slot %2d -> HRESULT 0x%08X  enum=%v\n", slot, uint32(r), enum != 0)
		lang.Free()
		query.Free()
		if enum != 0 {
			com.COMObject(enum).Release()
		}
	}
}
