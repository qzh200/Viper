//go:build windows

package wmi

import (
	"os"
	"testing"
)

// The client talks to the real WMI repository of the machine. The tests
// below use the standard namespace, which every machine has, so that they
// do not need the rights of an administrator and do not depend on the
// hardware of the vendor.
//
// The tests share one connection, because opening it costs a second on
// this machine and the tests would otherwise spend most of their time
// waiting for it.
var shared *Client

// processors holds the paths of the processors of the machine. The list
// is asked for once, because the query of the namespace takes a second
// on this machine.
var processors []string

func TestMain(m *testing.M) {
	client, err := Connect(NamespaceCIM)
	if err == nil {
		shared = client
		paths, err := client.Instances("Win32_Processor")
		if err == nil {
			processors = paths
		}
	}
	code := m.Run()
	if shared != nil {
		shared.Close()
	}
	os.Exit(code)
}

// connection returns the shared connection, or skips the test when the
// namespace is not reachable.
func connection(t *testing.T) *Client {
	t.Helper()
	if shared == nil {
		t.Skip("the namespace is not reachable")
	}
	return shared
}

// processorPaths returns the paths of the processors of the machine.
func processorPaths(t *testing.T) []string {
	t.Helper()
	if len(processors) == 0 {
		t.Skip("the processors cannot be listed")
	}
	return processors
}

func TestConnectToTheStandardNamespace(t *testing.T) {
	client, err := Connect(NamespaceCIM)
	if err != nil {
		t.Skipf("the namespace is not reachable: %v", err)
	}
	defer client.Close()
}

func TestInstancesOfTheStandardNamespace(t *testing.T) {
	client := connection(t)

	paths, err := client.Instances("Win32_Processor")
	if err != nil {
		t.Fatalf("the processors cannot be listed: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("the machine has no processor in the namespace")
	}
	for _, path := range paths {
		if path == "" {
			t.Error("the list carries an empty path")
		}
	}
}

func TestReadPropertyOfTheStandardNamespace(t *testing.T) {
	client := connection(t)
	paths := processorPaths(t)

	value, err := client.ReadProperty(paths[0], "MaxClockSpeed")
	if err != nil {
		t.Fatalf("the property cannot be read: %v", err)
	}
	if value == 0 {
		t.Error("the processor reports the highest clock as zero")
	}
}

func TestReadPropertyOfAPropertyThatIsNotThere(t *testing.T) {
	client := connection(t)
	paths := processorPaths(t)

	if _, err := client.ReadProperty(paths[0], "NoSuchProperty"); err == nil {
		t.Error("a property that is not there was read")
	}
}

func TestTheClientRefusesTheCallsAfterItIsClosed(t *testing.T) {
	client, err := Connect(NamespaceCIM)
	if err != nil {
		t.Skipf("the namespace is not reachable: %v", err)
	}
	client.Close()

	if _, err := client.Instances("Win32_Processor"); err == nil {
		t.Error("the closed client still lists the instances")
	}
	if _, err := client.Call("Win32_Processor", "GetOwner", "path", nil); err == nil {
		t.Error("the closed client still calls a method")
	}
	if _, err := client.ReadProperty("path", "Name"); err == nil {
		t.Error("the closed client still reads a property")
	}
}

func TestCloseTwice(t *testing.T) {
	client, err := Connect(NamespaceCIM)
	if err != nil {
		t.Skipf("the namespace is not reachable: %v", err)
	}
	client.Close()
	client.Close()
}
