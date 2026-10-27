//go:build tools

// Command kbtimeout looks for the query that reads the automatic switch
// off of the keyboard backlight.
//
// The setter puts the timeout into the action class, the getter of that
// class does not answer with the state, so the candidates are swept.
package main

import (
	"fmt"
	"os"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	candidates := []uint64{
		0x00088401,
		0x00088400,
		0x00088402,
		0x00088403,
		0x00008401,
		0x000108401,
		0x000088401,
		0x000888401,
		0x02088401,
		0x00088421,
		0x00088430,
		0x0008843F,
		0x0008841E,
		0x10088401,
		0x000C8401,
		0x00048401,
	}

	fmt.Println("=== action class GetFunction ===")
	for _, query := range candidates {
		outputs, err := client.Call(hardware.ClassAction, "GetFunction", path, map[string]wmi.Value{
			"uiInput": wmi.Uint32(uint32(query)),
		})
		if err != nil {
			fmt.Printf("  0x%09X -> %v\n", query, err)
			continue
		}
		value, ok := outputs["uiOutput"]
		if !ok {
			fmt.Printf("  0x%09X -> no output\n", query)
			continue
		}
		raw := value.UInt64()
		fmt.Printf("  0x%09X -> 0x%08X  byte4=%d byte5=%d byte6=%d byte7=%d\n",
			query, raw, (raw>>32)&0xFF, (raw>>40)&0xFF, (raw>>48)&0xFF, (raw>>56)&0xFF)
	}

	fmt.Println("\n=== gaming class candidates ===")
	for _, query := range []uint32{0, 1, 2, 132, 0x8401, 0x88401} {
		outputs, err := client.Call(hardware.ClassGaming, "GetGamingProfileSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint32(query),
		})
		if err != nil {
			fmt.Printf("  profileSetting(0x%X) -> %v\n", query, err)
			continue
		}
		value, ok := outputs["gmOutput"]
		if !ok {
			continue
		}
		fmt.Printf("  profileSetting(0x%X) -> 0x%08X\n", query, value.UInt64())
	}
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
		fmt.Println("no action instance:", err)
		os.Exit(1)
	}
	return client, paths[0]
}
