//go:build tools

// Command fanprobe reveals how the fan behaviour is encoded.
//
// The written word and the word the provider reports back are not the
// same, so every mode is applied and the answers are compared.
package main

import (
	"fmt"
	"os"
	"strings"

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

	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		fmt.Println("wmi:", err)
		os.Exit(1)
	}
	defer client.Close()

	paths, err := client.Instances(hardware.ClassGaming)
	if err != nil || len(paths) == 0 {
		fmt.Println("no instance:", err)
		os.Exit(1)
	}
	path := paths[0]

	type mode struct {
		name  string
		cpu   uint64
		gpu   uint64
		speed uint64
	}
	modes := []mode{
		{"auto", 1, 1, 0},
		{"max", 2, 2, 100},
		{"custom 40", 3, 3, 40},
		{"cpu custom 70", 3, 1, 70},
	}

	for _, m := range modes {
		word := uint64(9) | (m.cpu << 16) | (m.gpu << 22)
		fmt.Printf("=== %s (word 0x%X) ===\n", m.name, word)
		if _, err := client.Call(hardware.ClassGaming, "SetGamingFanBehavior", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(word),
		}); err != nil {
			fmt.Println("  behaviour ->", err)
		}
		if m.speed > 0 {
			for _, index := range []uint64{1, 4, 2} {
				if _, err := client.Call(hardware.ClassGaming, "SetGamingFanSpeed", path, map[string]wmi.Value{
					"gmInput": wmi.Uint64(m.speed<<8 | index),
				}); err != nil {
					fmt.Printf("  speed %d -> %v\n", index, err)
				}
			}
		}
		for query := uint32(0); query <= 5; query++ {
			value, err := readRaw(client, path, "GetGamingFanBehavior", query)
			if err != nil {
				continue
			}
			fmt.Printf("  behaviour(%d) = %s\n", query, value)
		}
		for index := uint32(1); index <= 5; index++ {
			value, err := readRaw(client, path, "GetGamingFanSpeed", index)
			if err != nil {
				continue
			}
			fmt.Printf("  speed(%d) = %s\n", index, value)
		}
		table, err := readRaw(client, path, "GetGamingFanTable", 0)
		if err == nil {
			fmt.Printf("  table = %s\n", table)
		}
		fmt.Println()
	}

	fmt.Println("=== restore auto ===")
	if _, err := client.Call(hardware.ClassGaming, "SetGamingFanBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(9 | 1<<16 | 1<<22),
	}); err != nil {
		fmt.Println("  failed:", err)
	}
	if _, err := client.Call(hardware.ClassGaming, "SetGamingFanTable", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1),
	}); err != nil {
		fmt.Println("  table ->", err)
	}
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
