//go:build tools

// Command cpuprobe asks the processor about its clocks.
//
// The standard namespace of Windows knows two numbers about the processor:
// the nominal one and the highest one it reaches. The method that asks the
// processor itself is the second one.
package main

import (
	"fmt"
	"os"

	"viper/internal/wmi"
)

func main() {
	client, err := wmi.Connect(wmi.NamespaceCIM)
	if err != nil {
		fmt.Println("wmi:", err)
		os.Exit(1)
	}
	defer client.Close()

	paths, err := client.Instances("Win32_Processor")
	if err != nil || len(paths) == 0 {
		fmt.Println("no processor:", err)
		os.Exit(1)
	}
	fmt.Println("path:", paths[0])

	for _, name := range []string{"MaxClockSpeed", "CurrentClockSpeed", "NumberOfCores", "NumberOfLogicalProcessors", "Name", "Description"} {
		value, err := client.ReadProperty(paths[0], name)
		if err != nil {
			fmt.Printf("  %-26s %v\n", name, err)
			continue
		}
		fmt.Printf("  %-26s %d\n", name, value)
	}

	for _, method := range []string{"GetMaxClockSpeed", "GetCurrentClockSpeed"} {
		outputs, err := client.Call("Win32_Processor", method, paths[0], nil)
		if err != nil {
			fmt.Printf("  %-26s %v\n", method, err)
			continue
		}
		for name, value := range outputs {
			fmt.Printf("  %-26s %s = %s\n", method, name, value.String())
		}
	}
}
