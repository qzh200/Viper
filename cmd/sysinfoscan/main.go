//go:build tools

// Command sysinfoscan walks the query space of GetGamingSysInfo and
// reports every category that returns a reading.
//
// The tool is how the performance profiles were told apart: the readings
// of the fan speeds separate the profiles clearly.
package main

import (
	"fmt"
	"os"
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

	paths, err := acpi.Instances(hardware.ClassGaming)
	if err != nil || len(paths) == 0 {
		fmt.Println("no instance:", err)
		os.Exit(1)
	}
	path := paths[0]

	fmt.Println("=== fan speeds per performance profile ===")
	for _, mode := range []uint64{0, 1, 4, 5, 6} {
		if _, err := acpi.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(mode<<8 | 0x0B),
		}); err != nil {
			fmt.Printf("value %d -> %v\n", mode, err)
			continue
		}
		time.Sleep(2000 * time.Millisecond)

		var cpu, gpu []uint64
		for round := 0; round < 3; round++ {
			cpu = append(cpu, shifted(acpi, path, 0x0201))
			gpu = append(gpu, shifted(acpi, path, 0x0601))
			time.Sleep(500 * time.Millisecond)
		}
		fmt.Printf("value %d: cpuFan=%v gpuFan=%v\n", mode, cpu, gpu)
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

// shifted reads a GetGamingSysInfo value and returns the reading itself.
func shifted(client *wmi.Client, path string, query uint32) uint64 {
	raw, err := readSysInfo(client, path, query)
	if err != nil {
		return 0
	}
	return (raw >> 8) & 0xFFFF
}

// readSysInfo reads one GetGamingSysInfo value.
func readSysInfo(client *wmi.Client, path string, query uint32) (uint64, error) {
	outputs, err := client.Call(hardware.ClassGaming, "GetGamingSysInfo", path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(query),
	})
	if err != nil {
		return 0, err
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("no gmOutput")
	}
	text := strings.TrimSpace(value.String())
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	return strconv.ParseUint(text, base, 64)
}
