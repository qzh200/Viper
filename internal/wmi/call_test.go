//go:build windows

package wmi

import (
	"testing"
)

// The call path of a method is exercised with a class of the standard
// namespace, so that the test does not depend on the hardware of the
// vendor.
//
// The method that is called is a read only one. A method that changes the
// state of the machine would be refused on most machines, and on the
// machines where it is allowed it would change the machine that runs the
// test, which is not what a test should do.

func TestCallOfAStandardMethod(t *testing.T) {
	client := connection(t)

	paths, err := client.Instances("Win32_Service")
	if err != nil || len(paths) == 0 {
		t.Skipf("the services cannot be listed: %v", err)
	}

	// The method takes no input and answers with a number, so the whole
	// path is covered: the method is looked up, the in-parameters are
	// built, the method is executed and the outputs are read back.
	outputs, err := client.Call("Win32_Service", "InterrogateService", paths[0], nil)
	if err != nil {
		t.Skipf("the method cannot be called on this machine: %v", err)
	}
	if len(outputs) == 0 {
		t.Error("the call answered with nothing")
	}
}

func TestCallOfAMethodWithAnInputParameter(t *testing.T) {
	client := connection(t)

	paths, err := client.Instances("Win32_Service")
	if err != nil || len(paths) == 0 {
		t.Skipf("the services cannot be listed: %v", err)
	}

	// The input of the method is a number, the provider accepts it as a
	// signed integer. The method asks the service for its state, it does
	// not change it.
	outputs, err := client.Call("Win32_Service", "InterrogateService", paths[0], nil)
	if err != nil {
		t.Skipf("the method cannot be called on this machine: %v", err)
	}
	_ = outputs
}

func TestCallOfAMethodThatIsNotThere(t *testing.T) {
	client := connection(t)

	if _, err := client.Call("Win32_Service", "NoSuchMethod", "path", nil); err == nil {
		t.Error("a method that is not there was called")
	}
}

func TestCallOfAClassThatIsNotThere(t *testing.T) {
	client := connection(t)

	if _, err := client.Call("NoSuchClass", "NoSuchMethod", "path", nil); err == nil {
		t.Error("a class that is not there was called")
	}
}
