//go:build windows

package wmi

import (
	"testing"
)

// The call path of a method is exercised with a class of the standard
// namespace, so that the test does not depend on the hardware of the
// vendor.

func TestCallOfAStandardMethod(t *testing.T) {
	client, err := Connect(NamespaceCIM)
	if err != nil {
		t.Skipf("the namespace is not reachable: %v", err)
	}
	defer client.Close()

	paths, err := client.Instances("Win32_OperatingSystem")
	if err != nil || len(paths) == 0 {
		t.Skipf("the operating system cannot be listed: %v", err)
	}

	// The method of the class that takes no input and answers with a
	// number.
	outputs, err := client.Call("Win32_OperatingSystem", "GetType", paths[0], nil)
	if err != nil {
		t.Skipf("the method cannot be called on this machine: %v", err)
	}
	if len(outputs) == 0 {
		t.Error("the call answered with nothing")
	}
}

func TestCallOfAMethodThatIsNotThere(t *testing.T) {
	client, err := Connect(NamespaceCIM)
	if err != nil {
		t.Skipf("the namespace is not reachable: %v", err)
	}
	defer client.Close()

	if _, err := client.Call("Win32_Processor", "NoSuchMethod", "path", nil); err == nil {
		t.Error("a method that is not there was called")
	}
}

func TestCallOfAClassThatIsNotThere(t *testing.T) {
	client, err := Connect(NamespaceCIM)
	if err != nil {
		t.Skipf("the namespace is not reachable: %v", err)
	}
	defer client.Close()

	if _, err := client.Call("NoSuchClass", "NoSuchMethod", "path", nil); err == nil {
		t.Error("a class that is not there was called")
	}
}
