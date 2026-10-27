//go:build windows

package wmi

import (
	"testing"
	"unsafe"

	"viper/internal/com"
)

// The values are put onto the in-parameters of a method and read back
// from the out-parameters. The objects of the standard namespace stand
// for them, so that the tests do not need the hardware of the vendor.

// instanceObject returns an instance of a class of the standard
// namespace.
//
// The properties are carried by the instances, the class object only
// describes them. The connection is shared, because opening it costs a
// second on this machine.
func instanceObject(t *testing.T, name string) com.COMObject {
	t.Helper()

	client := connection(t)

	path, err := com.NewBSTR(name)
	if err != nil {
		t.Fatal(err)
	}
	defer path.Free()

	var object uintptr
	r, _ := client.services.Call(6,
		uintptr(path), 0, 0,
		uintptr(unsafe.Pointer(&object)),
		0,
	)
	if int32(r) < 0 {
		t.Skipf("the class %s is not there: HRESULT 0x%08X", name, uint32(r))
	}
	class := com.COMObject(object)
	// The class object is not released here, it belongs to the
	// connection of the test, which is released with it.
	com.SetProxyBlanket(class)

	// The instances of the class carry the properties.
	query, err := com.NewBSTR("SELECT * FROM " + name)
	if err != nil {
		t.Fatal(err)
	}
	defer query.Free()

	lang, err := com.NewBSTR("WQL")
	if err != nil {
		t.Fatal(err)
	}
	defer lang.Free()

	var enumerator uintptr
	r, _ = client.services.Call(20,
		uintptr(lang),
		uintptr(query),
		0x30, // WBEM_FLAG_RETURN_IMMEDIATELY | WBEM_FLAG_FORWARD_ONLY
		0,    // ppCallResult
		uintptr(unsafe.Pointer(&enumerator)),
		0, // pFlags
	)
	if int32(r) < 0 {
		t.Skipf("the instances of %s cannot be listed: HRESULT 0x%08X", name, uint32(r))
	}
	enum := com.COMObject(enumerator)
	// The enumerator is not released here, it belongs to the connection
	// of the test, which is released with it.
	com.SetProxyBlanket(enum)

	var instance uintptr
	var returned uint32
	r, _ = enum.Call(4,
		uintptr(^uint64(0)),
		1,
		uintptr(unsafe.Pointer(&instance)),
		uintptr(unsafe.Pointer(&returned)),
	)
	if int32(r) < 0 || instance == 0 {
		t.Skipf("the instances of %s cannot be read: HRESULT 0x%08X", name, uint32(r))
	}
	object2 := com.COMObject(instance)
	// The instance is not released here, it belongs to the connection
	// of the test, which is released with it.
	com.SetProxyBlanket(object2)
	return object2
}

// processorInstance is the instance of the processor of the machine.
//
// The instance is asked for once, because the query of the namespace
// takes a second on this machine. The object belongs to the connection
// of the test, it is released with it.
var processorInstance com.COMObject

// sharedProcessorInstance returns the instance of the processor of the
// machine.
//
// The instance is asked for once, because the query of the namespace
// takes a second on this machine. The object belongs to the connection
// of the test, it is released with it.
func sharedProcessorInstance(t *testing.T) com.COMObject {
	t.Helper()
	if processorInstance == 0 {
		processorInstance = instanceObject(t, "Win32_Processor")
	}
	return processorInstance
}

func TestReadValueOfAProperty(t *testing.T) {
	instance := sharedProcessorInstance(t)

	value, err := readValue(instance, "MaxClockSpeed")
	if err != nil {
		t.Skipf("the property cannot be read on this machine: %v", err)
	}
	if value.Kind != KindUint64 {
		t.Errorf("the property has the kind %d, want the number", value.Kind)
	}
	if value.UInt64() == 0 {
		t.Error("the processor reports the highest clock as zero")
	}
}

func TestReadValueOfAText(t *testing.T) {
	instance := sharedProcessorInstance(t)

	value, err := readValue(instance, "Manufacturer")
	if err != nil {
		t.Skipf("the property cannot be read on this machine: %v", err)
	}
	if value.Kind != KindString {
		t.Errorf("the property has the kind %d, want the text", value.Kind)
	}
}

func TestReadValueOfAPropertyThatIsNotThere(t *testing.T) {
	instance := sharedProcessorInstance(t)

	if _, err := readValue(instance, "NoSuchProperty"); err == nil {
		t.Error("a property that is not there was read")
	}
}

func TestPutValueOntoAnObject(t *testing.T) {
	// The in-parameters of a method are an object like any other, so a
	// value can be put onto one. The value is read back afterwards, a
	// write that is silently dropped has to be caught.
	instance := sharedProcessorInstance(t)

	if err := putValue(instance, "Caption", String("a name")); err != nil {
		t.Skipf("the value cannot be put onto the object on this machine: %v", err)
	}

	value, err := readValue(instance, "Caption")
	if err != nil {
		t.Fatalf("the value cannot be read back: %v", err)
	}
	if value.Kind != KindString {
		t.Errorf("the value has the kind %d, want the text", value.Kind)
	}
	if value.Str != "a name" {
		t.Errorf("the value reads back as %q, want %q", value.Str, "a name")
	}
}

func TestPutValueOfANumber(t *testing.T) {
	instance := sharedProcessorInstance(t)

	if err := putValue(instance, "MaxClockSpeed", Uint64(4000)); err != nil {
		t.Skipf("the value cannot be put onto the object on this machine: %v", err)
	}

	value, err := readValue(instance, "MaxClockSpeed")
	if err != nil {
		t.Fatalf("the value cannot be read back: %v", err)
	}
	if value.UInt64() != 4000 {
		t.Errorf("the value reads back as %d, want 4000", value.UInt64())
	}
}

func TestPutValueOfABuffer(t *testing.T) {
	// The backlight setter takes its input as a buffer of bytes.
	instance := sharedProcessorInstance(t)

	if err := putValue(instance, "Caption", Bytes([]byte{1, 2, 3, 4})); err != nil {
		t.Skipf("the buffer cannot be put onto the object on this machine: %v", err)
	}
}

func TestPutValueOfAPropertyThatIsNotThere(t *testing.T) {
	// A property that the object does not carry cannot be written, the
	// provider refuses it.
	instance := sharedProcessorInstance(t)

	if err := putValue(instance, "NoSuchProperty", String("a name")); err == nil {
		t.Error("a property that is not there was written")
	}
}
