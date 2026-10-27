//go:build tools

// Command profileprobe looks for the meaning of the profile settings of
// the provider.
//
// The provider answers with a small number for every query. The tool
// reads all of them, changes a setting that is known and reads them again,
// so the query that belongs to the setting becomes visible.
package main

import (
	"fmt"
	"os"
	"time"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	fmt.Println("=== baseline ===")
	before := collect(client, path)
	print(before)

	fmt.Println("\n=== change the performance profile to performance ===")
	if _, err := client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(4<<8 | 0x0B),
	}); err != nil {
		fmt.Println("  write failed:", err)
	}
	time.Sleep(time.Second)

	after := collect(client, path)
	fmt.Println("=== differences ===")
	for key, value := range after {
		if before[key] != value {
			fmt.Printf("  %-22s %s -> %s\n", key, before[key], value)
		}
	}

	fmt.Println("\n=== restore the balanced profile ===")
	if _, err := client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1<<8 | 0x0B),
	}); err != nil {
		fmt.Println("  write failed:", err)
	}
	time.Sleep(time.Second)
	restored := collect(client, path)
	for key, value := range restored {
		if before[key] != value {
			fmt.Printf("  %-22s %s -> %s\n", key, before[key], value)
		}
	}
}

// collect reads every candidate of the profile settings.
func collect(client *wmi.Client, path string) map[string]string {
	out := make(map[string]string)
	for query := uint32(0); query <= 0x20; query++ {
		if value, err := readRaw(client, path, "GetGamingProfileSetting", query); err == nil {
			out[fmt.Sprintf("profileSetting(%d)", query)] = value
		}
	}
	for query := uint32(0); query <= 0x20; query++ {
		if value, err := readRaw(client, path, "GetGamingProfile", query); err == nil {
			out[fmt.Sprintf("profile(%d)", query)] = value
		}
	}
	return out
}

// print shows the collected readings.
func print(values map[string]string) {
	for key, value := range values {
		fmt.Printf("  %-22s %s\n", key, value)
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
	value, ok := outputs["gmOutput"]
	if !ok {
		return "", fmt.Errorf("no output")
	}
	return value.String(), nil
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
