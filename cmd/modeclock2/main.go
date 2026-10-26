//go:build windows

// Command modeclock2 measures the processor clock under load in each of the
// operation mode encodings that the provider accepts.
//
// Under a sustained load the clock follows the profile limits, so the
// fastest encoding is the turbo profile.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
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
		time.Sleep(2000 * time.Millisecond)

		mode, _ := readMode(acpi, path)
		clock := clockUnderLoad(cim)
		fan := fanSpeed(acpi, path)
		fmt.Printf("value %d -> mode=0x%X  cpuClockUnderLoad=%d MHz  cpuFan=%d RPM\n",
			value, mode, clock, fan)
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

// clockUnderLoad spins the processor and reports the clock it reached.
func clockUnderLoad(client *wmi.Client) uint64 {
	paths, err := client.Instances("Win32_Processor")
	if err != nil || len(paths) == 0 {
		return 0
	}

	stop := make(chan struct{})
	for worker := 0; worker < runtime.NumCPU(); worker++ {
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
					burn()
				}
			}
		}()
	}

	var peak uint64
	for i := 0; i < 8; i++ {
		time.Sleep(400 * time.Millisecond)
		value, err := client.ReadProperty(paths[0], "CurrentClockSpeed")
		if err == nil && value > peak {
			peak = value
		}
	}
	close(stop)
	return peak
}

// burn keeps the processor busy.
func burn() {
	x := 1
	for i := 0; i < 200000; i++ {
		x = x*3 + i
	}
	if x == 0 {
		fmt.Println(x)
	}
}

// fanSpeed reads the current CPU fan speed.
func fanSpeed(client *wmi.Client, path string) uint64 {
	outputs, err := client.Call(hardware.ClassGaming, "GetGamingSysInfo", path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(0x0201),
	})
	if err != nil {
		return 0
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0
	}
	text := strings.TrimSpace(value.String())
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	n, _ := strconv.ParseUint(text, base, 64)
	return (n >> 8) & 0xFFFF
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
