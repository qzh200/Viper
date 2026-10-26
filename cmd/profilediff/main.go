//go:build windows

// Command profilediff finds the bits that follow the performance profile.
//
// It scans the profile getters twice, once in the balanced profile and once
// in the performance profile, and reports the queries whose value changed.
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

	methods := []string{"GetGamingProfile", "GetGamingProfileSetting", "GetGamingMiscSetting"}

	fmt.Println("=== balanced profile ===")
	balanced := scan(client, path, methods)
	for _, line := range balanced {
		fmt.Println(" ", line)
	}

	fmt.Println()
	fmt.Println("=== switching to performance ===")
	if err := backend.SetPowerMode(hardware.PowerPerformance); err != nil {
		fmt.Println("  failed:", err)
	} else {
		mode, _ := backend.PowerMode()
		fmt.Println("  mode is now", mode)
		performance := scan(client, path, methods)
		fmt.Println("=== differences ===")
		for key, value := range performance {
			if balanced[key] != value {
				fmt.Printf("  %s: %s -> %s\n", key, balanced[key], value)
			}
		}
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

// scan reads every query of every method and keeps the non zero results.
func scan(client *wmi.Client, path string, methods []string) map[string]string {
	out := make(map[string]string)
	for _, method := range methods {
		for query := 0; query < 256; query++ {
			value, err := call(client, path, method, wmi.Uint32(uint32(query)))
			if err != nil || value == "" {
				continue
			}
			out[fmt.Sprintf("%s(%d)", method, query)] = value
		}
	}
	return out
}

// call invokes a method and renders the raw output value.
func call(client *wmi.Client, path, method string, input wmi.Value) (string, error) {
	outputs, err := client.Call(hardware.ClassGaming, method, path, map[string]wmi.Value{
		"gmInput": input,
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
		text := strings.TrimSpace(value.String())
		if n, err := strconv.ParseUint(text, 0, 64); err == nil {
			if n == 0 {
				continue
			}
			text = fmt.Sprintf("0x%X", n)
		} else {
			continue
		}
		parts = append(parts, name+"="+text)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("empty")
	}
	return strings.Join(parts, " "), nil
}
