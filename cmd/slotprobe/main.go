//go:build windows

// Command slotprobe discovers the vtable slot numbers of the WMI COM
// interfaces (IWbemServices, IWbemClassObject, IEnumWbemClassObject).
//
// It is a development tool: the constants it produces are pasted into
// internal/wmi/slots.go. It works entirely against the unprivileged
// root\CIMV2 and root\DEFAULT namespaces, so it needs no admin rights.
//
// Note that WMI namespace names are case sensitive on this machine: the
// repository exposes CIMV2, DEFAULT and WMI in upper case, and asking for
// "root\cimv2" is rejected with WBEM_E_INVALID_NAMESPACE.
package main

import (
	"fmt"
	"unsafe"

	"viper/internal/com"
)

func hr32(r uintptr) uint32 { return uint32(r) }

func main() {
	// Try both apartment models: WMI's IWbemLocator is apartment-threaded,
	// so a single-threaded apartment may be required.
	for _, apt := range []struct {
		name string
		id   uint32
	}{
		{"MTA", com.ApartmentMultiThreaded},
		{"STA", com.ApartmentSingleThreaded},
	} {
		if err := com.InitCOMApartment(apt.id); err != nil {
			fmt.Printf("--- apartment %s: InitCOM failed: %v\n", apt.name, err)
			continue
		}
		fmt.Printf("--- apartment %s ---\n", apt.name)
		probeOnce()
		com.UninitCOM()
	}
}

// probeOnce runs the full discovery sequence once.
func probeOnce() {
	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		fmt.Println("  CoCreateInstance:", err)
		return
	}
	defer locator.Release()

	if err := com.InitCOMSecurity(); err != nil {
		fmt.Println("  InitCOMSecurity:", err)
	}

	var ns com.COMObject
	for _, candidate := range []string{"root\\CIMV2", "root\\WMI", "root\\DEFAULT"} {
		o, err := connect(locator, candidate)
		if err != nil {
			fmt.Printf("  %-16s -> %v\n", candidate, err)
			continue
		}
		fmt.Printf("  %-16s -> OK\n", candidate)
		ns = o
		break
	}
	if ns.IsNil() {
		fmt.Println("  no namespace could be opened")
		return
	}
	com.SetProxyBlanket(ns)
	fmt.Println("  connected")

	probeServices(ns)
	probeClassObject(ns)
}

// probeBSTR verifies that a BSTR allocated by oleaut32 round-trips.
func probeBSTR() {
	b, err := com.NewBSTR("root\\CIMV2")
	if err != nil {
		fmt.Println("BSTR alloc failed:", err)
		return
	}
	defer b.Free()
	got := b.String()
	fmt.Printf("BSTR round-trip: %q (len=%d) match=%v\n", got, len(got), got == "root\\CIMV2")
}

// connect performs IWbemLocator::ConnectServer (vtable slot 3).
//
// ConnectServer takes nine parameters:
//
//	strServer, strNamespacePath, strLocale, strUser, strPassword,
//	lSecurityFlags, strAuthority, ppCtx, ppLoc
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
		return 0, fmt.Errorf("ConnectServer failed: HRESULT 0x%08X", hr32(r))
	}
	return com.COMObject(services), nil
}

// probeServices identifies ExecQuery, GetObject and ExecMethod on
// IWbemServices by matching the distinctive WMI error codes each one
// returns for a deliberately invalid argument.
func probeServices(services com.COMObject) {
	fmt.Println("\n=== IWbemServices ===")

	fmt.Println("-- ExecQuery candidates (expect WBEM_E_INVALID_QUERY 0x80041017) --")
	for slot := 3; slot <= 30; slot++ {
		lang, _ := com.NewBSTR("WQL")
		query, _ := com.NewBSTR("SELECT FROM")
		var enum uintptr
		r, _ := services.Call(slot,
			uintptr(lang), uintptr(query),
			0x30,
			uintptr(unsafe.Pointer(&enum)),
			0,
		)
		fmt.Printf("  slot %2d -> HRESULT 0x%08X  enum=%v\n", slot, hr32(r), enum != 0)
		lang.Free()
		query.Free()
		if enum != 0 {
			com.COMObject(enum).Release()
		}
	}

	fmt.Println("-- GetObject candidates (expect WBEM_E_NOT_FOUND 0x80041002) --")
	for slot := 3; slot <= 30; slot++ {
		path, _ := com.NewBSTR("\\\\bogus\\path")
		var obj uintptr
		r, _ := services.Call(slot,
			uintptr(path),
			0,
			0,
			uintptr(unsafe.Pointer(&obj)),
			0,
		)
		fmt.Printf("  slot %2d -> HRESULT 0x%08X  obj=%v\n", slot, hr32(r), obj != 0)
		path.Free()
		if obj != 0 {
			com.COMObject(obj).Release()
		}
	}

	fmt.Println("-- ExecMethod candidates (expect WBEM_E_METHOD_NOT_FOUND 0x8004101A) --")
	for slot := 3; slot <= 30; slot++ {
		objPath, _ := com.NewBSTR("StdRegProv")
		method, _ := com.NewBSTR("NoSuchMethod")
		var out uintptr
		r, _ := services.Call(slot,
			uintptr(objPath), uintptr(method),
			0,
			0,
			uintptr(unsafe.Pointer(&out)),
			0,
		)
		fmt.Printf("  slot %2d -> HRESULT 0x%08X  out=%v\n", slot, hr32(r), out != 0)
		objPath.Free()
		method.Free()
		if out != 0 {
			com.COMObject(out).Release()
		}
	}
}

// probeClassObject identifies Get, Put, GetMethod, SpawnInstance and
// GetNames on IWbemClassObject.
func probeClassObject(services com.COMObject) {
	fmt.Println("\n=== IWbemClassObject ===")

	// Grab the Win32_OperatingSystem class object.
	path, _ := com.NewBSTR("Win32_OperatingSystem")
	var cls uintptr
	r, _ := services.Call(6,
		uintptr(path), 0, 0, uintptr(unsafe.Pointer(&cls)), 0,
	)
	path.Free()
	if com.HRESULT(r).Failed() || cls == 0 {
		fmt.Printf("could not obtain class object: HRESULT 0x%08X\n", hr32(r))
		return
	}
	class := com.COMObject(cls)
	defer class.Release()

	fmt.Println("-- Get candidates (expect WBEM_E_NOT_FOUND 0x80041002) --")
	for slot := 3; slot <= 26; slot++ {
		name, _ := com.NewBSTR("NoSuchProperty")
		var v com.VARIANT
		v.Init()
		var flavor int32
		r, _ := class.Call(slot,
			uintptr(name), 0,
			uintptr(unsafe.Pointer(&v)),
			uintptr(unsafe.Pointer(&flavor)),
			0,
		)
		fmt.Printf("  slot %2d -> HRESULT 0x%08X  vt=0x%04X\n", slot, hr32(r), v.Type())
		v.Clear()
		name.Free()
	}

	fmt.Println("-- Put candidates (expect WBEM_E_NOT_FOUND 0x80041002) --")
	for slot := 3; slot <= 26; slot++ {
		name, _ := com.NewBSTR("NoSuchProperty")
		v := com.NewVariantI4(1)
		r, _ := class.Call(slot,
			uintptr(name), 0,
			uintptr(unsafe.Pointer(&v)),
			0,
		)
		fmt.Printf("  slot %2d -> HRESULT 0x%08X\n", slot, hr32(r))
		name.Free()
	}

	fmt.Println("-- GetMethod candidates (expect WBEM_E_NOT_FOUND 0x80041002) --")
	for slot := 3; slot <= 26; slot++ {
		name, _ := com.NewBSTR("NoSuchMethod")
		var inDef, outDef uintptr
		r, _ := class.Call(slot,
			uintptr(name), 0,
			uintptr(unsafe.Pointer(&inDef)),
			uintptr(unsafe.Pointer(&outDef)),
		)
		fmt.Printf("  slot %2d -> HRESULT 0x%08X  inDef=%v outDef=%v\n", slot, hr32(r), inDef != 0, outDef != 0)
		name.Free()
		if inDef != 0 {
			com.COMObject(inDef).Release()
		}
		if outDef != 0 {
			com.COMObject(outDef).Release()
		}
	}

	fmt.Println("-- SpawnInstance candidates (expect S_OK and non-nil object) --")
	for slot := 3; slot <= 26; slot++ {
		var obj uintptr
		r, _ := class.Call(slot, 0, uintptr(unsafe.Pointer(&obj)))
		fmt.Printf("  slot %2d -> HRESULT 0x%08X  obj=%v\n", slot, hr32(r), obj != 0)
		if obj != 0 {
			com.COMObject(obj).Release()
		}
	}

	fmt.Println("-- GetNames candidates (expect S_OK) --")
	for slot := 3; slot <= 26; slot++ {
		var names, quals uintptr
		r, _ := class.Call(slot, 0, uintptr(unsafe.Pointer(&names)), uintptr(unsafe.Pointer(&quals)))
		fmt.Printf("  slot %2d -> HRESULT 0x%08X  names=%v\n", slot, hr32(r), names != 0)
		if names != 0 {
			com.COMObject(names).Release()
		}
		if quals != 0 {
			com.COMObject(quals).Release()
		}
	}
}
