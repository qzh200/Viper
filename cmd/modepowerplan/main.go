//go:build windows

// Command modepowerplan switches the operation mode and reports the
// Windows power plan processor settings for each encoding.
//
// Acer drives the performance profiles through the embedded controller, but
// the firmware usually also relaxes the processor limits in the active
// power plan, which makes the plan a convenient second signal.
package main

import (
	"fmt"
	"os"
	"os/exec"
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

	for _, value := range []uint64{0, 1, 4, 5, 6} {
		if _, err := acpi.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(value<<8 | 0x0B),
		}); err != nil {
			fmt.Printf("value %d -> %v\n", value, err)
			continue
		}
		time.Sleep(1500 * time.Millisecond)

		mode, _ := readMode(acpi, path)
		fmt.Printf("\n===== value %d -> mode=0x%X =====\n", value, mode)
		printPlan()
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

// printPlan prints the processor settings of the active power plan.
func printPlan() {
	out, err := exec.Command("powercfg", "/query", "SCHEME_CURRENT", "SUB_PROCESSOR").CombinedOutput()
	if err != nil {
		fmt.Println("  powercfg failed:", err)
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		text := strings.TrimRight(line, "\r")
		if strings.TrimSpace(text) == "" {
			continue
		}
		fmt.Println("  ", text)
	}
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
	text := strings.TrimSpace(value.String())
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	return strconv.ParseUint(text, base, 64)
}
