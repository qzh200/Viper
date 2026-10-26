//go:build tools

// Command actionprobe searches the generic ACPI action class for the
// keyboard lighting state.
//
// The Acer gaming class reports the brightness through GetGamingRgbKb but
// nothing about the colour or the effect, so the generic action class is
// scanned for the remaining properties.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	fmt.Println("=== action class readings ===")
	for query := uint32(0); query <= 0x40; query++ {
		value, err := readAction(client, path, query)
		if err != nil {
			continue
		}
		fmt.Printf("  GetFunction(0x%02X) = %s\n", query, value)
	}

	fmt.Println()
	fmt.Println("=== after switching to the wave effect ===")
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(3<<16 | 5<<8 | 1),
	}); err != nil {
		fmt.Println("  effect ->", err)
	}
	for query := uint32(0); query <= 0x40; query++ {
		value, err := readAction(client, path, query)
		if err != nil {
			continue
		}
		fmt.Printf("  GetFunction(0x%02X) = %s\n", query, value)
	}
}

// readAction reads one value of the generic action class.
func readAction(client *wmi.Client, path string, query uint32) (string, error) {
	outputs, err := client.Call(hardware.ClassAction, "GetFunction", path, map[string]wmi.Value{
		"uiInput": wmi.Uint32(query),
	})
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(outputs))
	for _, name := range []string{"uiOutput", "gmOutput"} {
		value, ok := outputs[name]
		if !ok {
			continue
		}
		parts = append(parts, name+"="+value.String())
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("no output")
	}
	return strings.Join(parts, " "), nil
}

// connect opens the ACPI namespace and returns the action instance path.
func connect() (*wmi.Client, string) {
	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		fmt.Println("wmi:", err)
		os.Exit(1)
	}
	paths, err := client.Instances(hardware.ClassAction)
	if err != nil || len(paths) == 0 {
		paths, err = client.Instances(hardware.ClassGaming)
		if err != nil || len(paths) == 0 {
			fmt.Println("no instance:", err)
			os.Exit(1)
		}
	}
	return client, paths[0]
}

// unused keeps strconv referenced for future probes.
var _ = strconv.Itoa
