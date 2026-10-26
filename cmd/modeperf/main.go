//go:build windows

// Command modeperf compares the live processor performance in each of the
// operation mode encodings that the provider accepts.
//
// The processor performance counter follows the boost behaviour, so it
// separates the profiles that the static clock speed cannot.
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

		// The firmware needs a moment before the new limits apply, and the
		// counter is noisy, so several samples are taken.
		time.Sleep(2500 * time.Millisecond)

		mode, _ := readMode(acpi, path)
		var samples []uint64
		for i := 0; i < 4; i++ {
			samples = append(samples, processorPerformance(cim))
			time.Sleep(400 * time.Millisecond)
		}
		fmt.Printf("value %d -> mode=0x%X  processorPerformance samples=%v\n",
			value, mode, samples)
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

// processorPerformance reads the live processor performance percentage.
func processorPerformance(client *wmi.Client) uint64 {
	paths, err := client.Instances("Win32_PerfFormattedData_Counters_ProcessorInformation")
	if err != nil || len(paths) == 0 {
		return 0
	}
	value, err := client.ReadProperty(paths[0], "PercentProcessorPerformance")
	if err != nil {
		return 0
	}
	return value
}

// processorLoad reads the processor load percentage.
func processorLoad(client *wmi.Client) uint64 {
	paths, err := client.Instances("Win32_Processor")
	if err != nil || len(paths) == 0 {
		return 0
	}
	value, err := client.ReadProperty(paths[0], "LoadPercentage")
	if err != nil {
		return 0
	}
	return value
}

// readMode reads the raw operation mode.
func readMode(client *wmi.Client, path string) (uint64, error) {
	outputs, err := client.Call(hardware.ClassGaming, "GetGamingMiscSetting", path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(0x0B),
	})
	if err != nil {
		return 0, err
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("no gmOutput")
	}
	return value.UInt64(), nil
}
