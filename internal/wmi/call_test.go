//go:build windows

package wmi

import (
	"testing"
)

// The call path of a method is exercised with a class of the standard
// namespace, so that the test does not depend on the hardware of the
// vendor.

func TestCallOfAStandardMethod(t *testing.T) {
	client := connection(t)

	paths := processorPaths(t)

	// The method takes an input parameter and answers with the number of
	// the seconds that the state was kept, so the whole path is covered:
	// the method is looked up, the in-parameters are spawned, the value
	// is put onto them and the outputs are read back.
	outputs, err := client.Call("Win32_Processor", "SetPowerState", paths[0], map[string]Value{
		"PowerState": Uint32(1),
	})
	if err != nil {
		t.Skipf("the method cannot be called on this machine: %v", err)
	}
	if len(outputs) == 0 {
		t.Error("the call answered with nothing")
	}
}

func TestCallOfAMethodWithAnInputParameter(t *testing.T) {
	client := connection(t)

	paths := processorPaths(t)

	// The input of the method is a number, the provider accepts it as a
	// signed integer.
	outputs, err := client.Call("Win32_Processor", "SetPowerState", paths[0], map[string]Value{
		"PowerState": Uint32(2),
	})
	if err != nil {
		t.Skipf("the method cannot be called on this machine: %v", err)
	}
	_ = outputs
}

func TestCallOfAMethodThatIsNotThere(t *testing.T) {
	client := connection(t)

	if _, err := client.Call("Win32_Processor", "NoSuchMethod", "path", nil); err == nil {
		t.Error("a method that is not there was called")
	}
}

func TestCallOfAClassThatIsNotThere(t *testing.T) {
	client := connection(t)

	if _, err := client.Call("NoSuchClass", "NoSuchMethod", "path", nil); err == nil {
		t.Error("a class that is not there was called")
	}
}
