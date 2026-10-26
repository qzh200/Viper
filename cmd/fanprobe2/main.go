//go:build tools

// Command fanprobe2 separates the fan groups.
//
// The CPU and the GPU fan are set to different modes and every query of
// GetGamingFanBehavior is read back, which shows which query belongs to
// which group.
package main

import (
	"fmt"
	"os"
	"strings"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	type setup struct {
		name string
		cpu  uint64
		gpu  uint64
	}
	for _, s := range []setup{
		{"cpu=auto gpu=max", 1, 2},
		{"cpu=max gpu=auto", 2, 1},
		{"cpu=custom gpu=auto", 3, 1},
		{"cpu=auto gpu=custom", 1, 3},
	} {
		word := uint64(9) | (s.cpu << 16) | (s.gpu << 22)
		fmt.Printf("=== %s (word 0x%X) ===\n", s.name, word)
		if _, err := client.Call(hardware.ClassGaming, "SetGamingFanBehavior", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(word),
		}); err != nil {
			fmt.Println("  failed:", err)
		}
		for query := uint32(0); query <= 7; query++ {
			value, err := readRaw(client, path, "GetGamingFanBehavior", query)
			if err != nil {
				continue
			}
			fmt.Printf("  behaviour(%d) = %s\n", query, value)
		}
		fmt.Println()
	}

	fmt.Println("=== restore auto ===")
	if _, err := client.Call(hardware.ClassGaming, "SetGamingFanBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(9 | 1<<16 | 1<<22),
	}); err != nil {
		fmt.Println("  failed:", err)
	}
}

// connect opens the ACPI namespace and returns the gaming instance path.
func connect() (*wmi.Client, string) {
	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		fmt.Println("wmi:", err)
		os.Exit(1)
	}
	paths, err := client.Instances(hardware.ClassGaming)
	if err != nil || len(paths) == 0 {
		fmt.Println("no instance:", err)
		os.Exit(1)
	}
	return client, paths[0]
}

// readRaw invokes a method and renders the raw output value.
func readRaw(client *wmi.Client, path, method string, query uint32) (string, error) {
	outputs, err := client.Call(hardware.ClassGaming, method, path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(query),
	})
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(outputs))
	for _, name := range []string{"gmOutput", "gmReturn"} {
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
