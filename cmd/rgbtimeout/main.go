//go:build tools

// Command rgbtimeout checks whether the flags byte of the keyboard
// backlight buffer carries the automatic switch off.
//
// The original service reports the state of the switch, so it is used as
// the reference while the buffer is written with different flags.
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	// With an argument the probe writes a single buffer, so that the
	// reference of the original service can be read between the writes.
	if len(os.Args) > 1 {
		flags := byte(0)
		fmt.Sscan(os.Args[1], &flags)
		level := 60
		if len(os.Args) > 2 {
			fmt.Sscan(os.Args[2], &level)
		}
		write(client, path, level, flags)
		fmt.Printf("  buffer   : %s\n", buffer(client, path))
		fmt.Printf("  level    : %s\n", raw(client, path, "GetGamingRgbKb", 1))
		return
	}

	fmt.Println("=== baseline ===")
	fmt.Printf("  buffer   : %s\n", buffer(client, path))
	fmt.Printf("  level    : %s\n", raw(client, path, "GetGamingRgbKb", 1))

	for _, flags := range []byte{0, 1, 2, 3, 4} {
		fmt.Printf("--- flags byte = %d ---\n", flags)
		write(client, path, 60, flags)
		fmt.Printf("  buffer   : %s\n", buffer(client, path))
		fmt.Printf("  level    : %s\n", raw(client, path, "GetGamingRgbKb", 1))
	}

	fmt.Println("\n=== restore ===")
	write(client, path, 100, 3)
	fmt.Printf("  buffer   : %s\n", buffer(client, path))
	fmt.Printf("  level    : %s\n", raw(client, path, "GetGamingRgbKb", 1))
}

// write sends a backlight buffer with the wanted level and flags.
func write(client *wmi.Client, path string, level int, flags byte) {
	data := make([]byte, 16)
	data[2] = byte(level)
	data[5] = byte(level)
	data[8] = flags
	data[9] = 1
	_, err := client.Call(hardware.ClassGaming, "SetGamingKBBacklight", path, map[string]wmi.Value{
		"gmInput": wmi.Bytes(data),
	})
	if err != nil {
		fmt.Println("  write failed:", err)
		return
	}
	if _, err := client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1 | uint64(level)<<8),
	}); err != nil {
		fmt.Println("  level write failed:", err)
	}
}

// raw invokes a getter that answers with a number.
func raw(client *wmi.Client, path, method string, query uint32) string {
	outputs, err := client.Call(hardware.ClassGaming, method, path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(query),
	})
	if err != nil {
		return "error: " + err.Error()
	}
	parts := make([]string, 0, 3)
	for _, name := range []string{"gmOutput", "gmReturn"} {
		if value, ok := outputs[name]; ok {
			parts = append(parts, name+"="+value.String())
		}
	}
	return strings.Join(parts, " ")
}

// buffer invokes a getter that answers with a byte array.
func buffer(client *wmi.Client, path string) string {
	outputs, err := client.Call(hardware.ClassGaming, "GetGamingKBBacklight", path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(1),
	})
	if err != nil {
		return "error: " + err.Error()
	}
	parts := make([]string, 0, 3)
	for _, name := range []string{"gmOutput", "gmReturn"} {
		value, ok := outputs[name]
		if !ok {
			continue
		}
		if value.Kind == wmi.KindBytes {
			parts = append(parts, name+"="+hex.EncodeToString(value.Bytes))
		} else {
			parts = append(parts, name+"="+value.String())
		}
	}
	return strings.Join(parts, " ")
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
