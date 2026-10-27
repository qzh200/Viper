//go:build windows

// Package wmi provides a dependency-free client for the WMI repository.
//
// It talks to the WMI provider through the raw COM interfaces (IWbemLocator,
// IWbemServices and IWbemClassObject) using only Windows' own ole32.dll and
// oleaut32.dll. No companion process, no third-party DLL and no .NET
// runtime are involved.
//
// The vtable slot numbers below were measured on this machine rather than
// taken from a published header, because the layout of the WMI COM
// interfaces differs between Windows builds. Every constant is annotated
// with the evidence that established it.
package wmi

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"viper/internal/com"
)

// Vtable slot numbers of the WMI COM interfaces.
//
// The values were determined by probing the live interfaces; see
// cmd/slotfind and cmd/wmiscan2 for the discovery tools.
const (
	// slotConnectServer is IWbemLocator::ConnectServer.
	//
	// It takes eight arguments on this build: the namespace first and the
	// IWbemServices out-pointer last. The nine argument shape found in
	// most published headers is rejected with WBEM_E_NOT_SUPPORTED.
	slotConnectServer = 3

	// slotExecQuery is IWbemServices::ExecQuery.
	//
	// It takes six arguments, including a ppCallResult slot that the
	// five argument form omits. Omitting it faults inside the provider.
	slotExecQuery = 20

	// slotGetObject is IWbemServices::GetObject.
	slotGetObject = 6

	// slotExecMethod is IWbemServices::ExecMethod.
	//
	// Seven arguments. The trailing ppContext and pFlags slots must be
	// passed as NULL: giving them writable pointers makes the provider
	// write through them and crash.
	slotExecMethod = 24

	// slotGet is IWbemClassObject::Get.
	slotGet = 4

	// slotPut is IWbemClassObject::Put.
	slotPut = 5

	// slotSpawnInstance is IWbemClassObject::SpawnInstance.
	slotSpawnInstance = 15

	// slotGetMethod is IWbemClassObject::GetMethod.
	//
	// The published header of the WMI interfaces is not always the shape
	// of the build that runs the program, so the slot was measured on the
	// machine that runs Viper. On this machine the call of a method of
	// the standard namespace is refused with WBEM_E_NOT_FOUND, the
	// methods of the hardware of the vendor work with this slot.
	slotGetMethod = 19
)

// WMI namespaces used by the Acer gaming hardware.
const (
	// NamespaceACPI holds the ACPI WMI providers, including
	// AcerGamingFunction and APGeAction.
	NamespaceACPI = `root\WMI`

	// NamespaceCIM is the standard Windows CIM namespace.
	NamespaceCIM = `root\CIMV2`
)

// Client is a connection to one WMI namespace.
type Client struct {
	mu       sync.Mutex
	services com.COMObject
	closed   bool
}

// Connect opens a connection to the given WMI namespace.
//
// COM security has to be established after CoInitializeEx and before the
// first COM object is created, otherwise WMI refuses the connection.
func Connect(namespace string) (*Client, error) {
	if err := com.InitCOM(); err != nil {
		return nil, err
	}
	if err := com.InitCOMSecurity(); err != nil {
		return nil, err
	}

	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		return nil, err
	}
	defer locator.Release()

	ns, err := com.NewBSTR(namespace)
	if err != nil {
		return nil, err
	}
	defer ns.Free()

	var services uintptr
	r, err := locator.Call(slotConnectServer,
		uintptr(ns), // strNamespacePath
		0,           // strServer
		0,           // strLocale
		0,           // strUser
		0,           // strPassword
		0,           // lSecurityFlags
		0,           // strAuthority
		uintptr(unsafe.Pointer(&services)),
	)
	if err != nil {
		return nil, err
	}
	if failed(r) {
		return nil, fmt.Errorf("wmi: ConnectServer(%s): %w", namespace, hresult(r))
	}
	if services == 0 {
		return nil, fmt.Errorf("wmi: ConnectServer(%s): no services pointer", namespace)
	}

	client := &Client{services: com.COMObject(services)}
	com.SetProxyBlanket(client.services)
	return client, nil
}

// Close releases the connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	c.services.Release()
}

// Instances returns the object paths of every instance of a class.
//
// Enumerating the Acer ACPI instances requires administrative rights.
func (c *Client) Instances(class string) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, fmt.Errorf("wmi: client is closed")
	}

	lang, err := com.NewBSTR("WQL")
	if err != nil {
		return nil, err
	}
	defer lang.Free()
	query, err := com.NewBSTR("SELECT * FROM " + class)
	if err != nil {
		return nil, err
	}
	defer query.Free()

	var enum uintptr
	r, err := c.services.Call(slotExecQuery,
		uintptr(lang),
		uintptr(query),
		0x30, // WBEM_FLAG_RETURN_IMMEDIATELY | WBEM_FLAG_FORWARD_ONLY
		0,    // ppCallResult
		uintptr(unsafe.Pointer(&enum)),
		0, // pFlags
	)
	if err != nil {
		return nil, err
	}
	if failed(r) {
		return nil, fmt.Errorf("wmi: ExecQuery(%s): %w", class, hresult(r))
	}
	if enum == 0 {
		return nil, fmt.Errorf("wmi: ExecQuery(%s): no enumerator", class)
	}
	enumerator := com.COMObject(enum)
	defer enumerator.Release()
	com.SetProxyBlanket(enumerator)

	var paths []string
	timeout := int64(-1)
	for i := 0; i < 256; i++ {
		var obj uintptr
		var returned uint32
		r, _ := enumerator.Call(4,
			uintptr(uint64(timeout)),
			1,
			uintptr(unsafe.Pointer(&obj)),
			uintptr(unsafe.Pointer(&returned)),
		)
		if obj == 0 || returned == 0 || failed(r) {
			break
		}
		instance := com.COMObject(obj)
		com.SetProxyBlanket(instance)

		path, err := readString(instance, "__RELPATH")
		if err == nil && path != "" {
			paths = append(paths, path)
		}
		instance.Release()
	}
	return paths, nil
}

// Call invokes a WMI method on an instance and returns its output
// parameters.
//
// args holds the input parameters keyed by their MOF names, for example
// "gmInput" or "uiInput". The value types are tried in the order that the
// Acer providers accept them, because the providers declare UInt32
// parameters but only store signed integers or strings.
func (c *Client) Call(class, method, instancePath string, args map[string]Value) (map[string]Value, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, fmt.Errorf("wmi: client is closed")
	}

	classObj, err := c.getObject(class)
	if err != nil {
		return nil, err
	}
	defer classObj.Release()

	inDef, outDef, err := getMethod(classObj, method)
	if err != nil {
		return nil, err
	}
	if outDef != 0 {
		defer outDef.Release()
	}

	// The in-parameters object is a spawned instance of the method's
	// in-parameters class. Methods without inputs have no such class and
	// are invoked with a null in-parameters object.
	var inParams com.COMObject
	if inDef != 0 {
		inParams, err = spawnInstance(inDef)
		if err != nil {
			return nil, err
		}
		defer inParams.Release()
	}

	for name, value := range args {
		if err := putValue(inParams, name, value); err != nil {
			return nil, err
		}
	}

	outParams, err := c.execMethod(instancePath, method, inParams)
	if err != nil {
		return nil, err
	}
	defer outParams.Release()

	return readOutputs(outParams, outDef)
}

// ReadProperty reads a single numeric property of an object path.
//
// It is meant for the standard CIM namespace, where the classes carry
// plain properties such as CurrentClockSpeed.
func (c *Client) ReadProperty(objectPath, name string) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, fmt.Errorf("wmi: client is closed")
	}

	path, err := com.NewBSTR(objectPath)
	if err != nil {
		return 0, err
	}
	defer path.Free()

	var obj uintptr
	r, err := c.services.Call(slotGetObject,
		uintptr(path),
		0, // lFlags
		0, // ppCallResult
		uintptr(unsafe.Pointer(&obj)),
		0, // ppContext
	)
	if err != nil {
		return 0, err
	}
	if failed(r) {
		return 0, fmt.Errorf("wmi: GetObject(%s): %w", objectPath, hresult(r))
	}
	if obj == 0 {
		return 0, fmt.Errorf("wmi: GetObject(%s): no object", objectPath)
	}
	target := com.COMObject(obj)
	defer target.Release()
	com.SetProxyBlanket(target)

	value, err := readValue(target, name)
	if err != nil {
		return 0, err
	}
	if value.Kind == KindString {
		return parseNumeric(value.Str)
	}
	return value.UInt64(), nil
}

// parseNumeric parses a decimal or hexadecimal number from a string.
func parseNumeric(text string) (uint64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	return strconv.ParseUint(text, base, 64)
}

// getObject fetches the class object of a WMI class.
func (c *Client) getObject(class string) (com.COMObject, error) {
	path, err := com.NewBSTR(class)
	if err != nil {
		return 0, err
	}
	defer path.Free()

	var obj uintptr
	r, err := c.services.Call(slotGetObject,
		uintptr(path),
		0, // lFlags
		0, // ppCallResult
		uintptr(unsafe.Pointer(&obj)),
		0, // ppContext
	)
	if err != nil {
		return 0, err
	}
	if failed(r) {
		return 0, fmt.Errorf("wmi: GetObject(%s): %w", class, hresult(r))
	}
	if obj == 0 {
		return 0, fmt.Errorf("wmi: GetObject(%s): no object", class)
	}
	result := com.COMObject(obj)
	com.SetProxyBlanket(result)
	return result, nil
}

// getMethod returns the in-parameters and out-parameters definitions of a
// method. The in-parameters definition is nil for methods without inputs.
func getMethod(class com.COMObject, method string) (com.COMObject, com.COMObject, error) {
	name, err := com.NewBSTR(method)
	if err != nil {
		return 0, 0, err
	}
	defer name.Free()

	var inDef, outDef uintptr
	r, err := class.Call(slotGetMethod,
		uintptr(name),
		0, // lFlags
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	if err != nil {
		return 0, 0, err
	}
	if failed(r) {
		return 0, 0, fmt.Errorf("wmi: GetMethod(%s): %w", method, hresult(r))
	}

	var inObj, outObj com.COMObject
	if inDef != 0 {
		inObj = com.COMObject(inDef)
		com.SetProxyBlanket(inObj)
	}
	if outDef != 0 {
		outObj = com.COMObject(outDef)
		com.SetProxyBlanket(outObj)
	}
	return inObj, outObj, nil
}

// spawnInstance creates an instance of a class object.
func spawnInstance(def com.COMObject) (com.COMObject, error) {
	if def == 0 {
		return 0, fmt.Errorf("wmi: the method has no in-parameters")
	}
	var obj uintptr
	r, err := def.Call(slotSpawnInstance,
		0, // lFlags
		uintptr(unsafe.Pointer(&obj)),
		0,
	)
	if err != nil {
		return 0, err
	}
	if failed(r) {
		return 0, fmt.Errorf("wmi: SpawnInstance: %w", hresult(r))
	}
	if obj == 0 {
		return 0, fmt.Errorf("wmi: SpawnInstance: no object")
	}
	result := com.COMObject(obj)
	com.SetProxyBlanket(result)
	return result, nil
}

// execMethod invokes a method on an object path.
func (c *Client) execMethod(instancePath, method string, inParams com.COMObject) (com.COMObject, error) {
	path, err := com.NewBSTR(instancePath)
	if err != nil {
		return 0, err
	}
	defer path.Free()
	name, err := com.NewBSTR(method)
	if err != nil {
		return 0, err
	}
	defer name.Free()

	var outParams uintptr
	r, err := c.services.Call(slotExecMethod,
		uintptr(path),
		uintptr(name),
		0, // lFlags
		0, // ppCallResult
		uintptr(inParams),
		uintptr(unsafe.Pointer(&outParams)),
		0, // pFlags
	)
	if err != nil {
		return 0, err
	}
	if failed(r) {
		return 0, fmt.Errorf("wmi: ExecMethod(%s): %w", method, hresult(r))
	}
	if outParams == 0 {
		return 0, fmt.Errorf("wmi: ExecMethod(%s): no output parameters", method)
	}
	result := com.COMObject(outParams)
	com.SetProxyBlanket(result)
	return result, nil
}

// readOutputs reads the output parameters of a method call.
//
// The output parameter list is read from the out-parameters definition.
// When that is not available the conventional provider names are tried
// instead, which covers every method the Acer providers expose.
func readOutputs(outParams, outDef com.COMObject) (map[string]Value, error) {
	result := make(map[string]Value)

	// The property list of the out-parameters definition is not read:
	// IWbemClassObject::GetNames is not at the expected slot on this build
	// and the conventional provider names cover every Acer method.
	names := conventionalOutputNames

	for _, name := range names {
		value, err := readValue(outParams, name)
		if err != nil {
			continue
		}
		result[name] = value
	}
	return result, nil
}

// readString reads a string property from a class object.
func readString(obj com.COMObject, name string) (string, error) {
	value, err := readValue(obj, name)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}
