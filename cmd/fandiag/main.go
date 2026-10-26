//go:build windows

// Command fandiag prints the raw values of the fan related provider
// methods, so that the duty cycle semantics can be checked against the
// physical behaviour of the machine.
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

	fmt.Println("instance:", path)
	fmt.Println()

	for _, query := range []uint32{0, 4} {
		raw, err := call(client, path, "GetGamingFanBehavior", wmi.Uint32(query))
		fmt.Printf("GetGamingFanBehavior(%d) = %s (err=%v)\n", query, raw, err)
	}

	for _, index := range []uint32{1, 2, 3, 4} {
		raw, err := call(client, path, "GetGamingFanSpeed", wmi.Uint32(index))
		fmt.Printf("GetGamingFanSpeed(%d) = %s (err=%v)\n", index, raw, err)
	}

	for _, method := range []string{"GetGamingFanTable", "GetGamingProfile"} {
		raw, err := call(client, path, method, wmi.Uint32(0))
		fmt.Printf("%s = %s (err=%v)\n", method, raw, err)
	}
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
	for name, value := range outputs {
		text := value.String()
		if n, err := strconv.ParseUint(strings.TrimPrefix(text, "0x"), 16, 64); err == nil {
			text = fmt.Sprintf("%s (0x%X)", text, n)
		}
		parts = append(parts, name+"="+text)
	}
	return strings.Join(parts, " "), nil
}
