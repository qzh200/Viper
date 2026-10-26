//go:build windows

// Command modeclock measures the processor clock speed in each of the
// operation mode encodings that the provider accepts.
//
// The provider keeps five values: zero, one, four, five and six. The
// fastest one is the turbo profile.
package main

import (
	"fmt"
	"os"
	"time"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	backend, err := hardware.NewBackend()
	if err != nil {
		fmt.Println("hardware:", err)
		os.Exit(1)
	}
	defer backend.Close()

	acpi, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		fmt.Println("wmi:", err)
		os.Exit(1)
	}
	defer acpi.Close()

	acpiPaths, err := acpi.Instances(hardware.ClassGaming)
	if err != nil || len(acpiPaths) == 0 {
		fmt.Println("no instance:", err)
		os.Exit(1)
	}
	path := acpiPaths[0]

	cim, err := wmi.Connect(wmi.NamespaceCIM)
	if err != nil {
		fmt.Println("wmi cim:", err)
		os.Exit(1)
	}
	defer cim.Close()

	for _, value := range []uint64{0, 1, 4, 5, 6} {
		if _, err := acpi.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(value<<8 | 0x0B),
		}); err != nil {
			fmt.Printf("value %d -> %v\n", value, err)
			continue
		}

		// Let the firmware settle before the reading.
		time.Sleep(1500 * time.Millisecond)

		mode, _ := readMode(acpi, path)
		clock, maxClock := processorClock(cim)
		fmt.Printf("value %d -> mode=0x%X  cpu clock=%d MHz (max %d MHz)\n",
			value, mode, clock, maxClock)
	}

	fmt.Println()
	fmt.Println("=== restore balanced ===")
	if err := backend.SetPowerMode(hardware.PowerBalanced); err != nil {
		fmt.Println("  failed:", err)
	} else {
		mode, _ := backend.PowerMode()
		fmt.Println("  mode is now", mode)
	}
}

// processorClock reads the current and maximum clock of the processor.
func processorClock(client *wmi.Client) (current, maximum uint64) {
	paths, err := client.Instances("Win32_Processor")
	if err != nil || len(paths) == 0 {
		return 0, 0
	}
	for _, path := range paths {
		value, err := client.ReadProperty(path, "CurrentClockSpeed")
		if err == nil {
			current = value
		}
		value, err = client.ReadProperty(path, "MaxClockSpeed")
		if err == nil {
			maximum = value
		}
	}
	return current, maximum
}

// readMode reads the raw operation mode.
func readMode(client *wmi.Client, path string) (uint64, error) {
	return client.ReadProperty(path, "gmOutput")
}
