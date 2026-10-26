//go:build tools

// Command activetoggle finds the call that switches the active lighting
// mode.
//
// The profile of the Acer lighting service keeps the active mode in
// ActiveMode, so every candidate setting is tried and the profile is read
// back afterwards.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

const profilePath = `C:\ProgramData\OEM\AcerLightingService\LightingProfile\LightingProfile.ini`

func main() {
	client, path := connect()

	fmt.Println("active mode at the start:", activeMode())

	fmt.Println()
	fmt.Println("=== miscellaneous settings ===")
	for id := uint64(0); id <= 20; id++ {
		if _, err := client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(id),
		}); err != nil {
			continue
		}
		time.Sleep(400 * time.Millisecond)
		if mode := activeMode(); mode != 0 {
			fmt.Printf("  setting %d -> ActiveMode=%d\n", id, mode)
		}
	}

	fmt.Println()
	fmt.Println("=== gaming profile ===")
	for _, value := range []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 16, 17, 18, 19, 20} {
		if _, err := client.Call(hardware.ClassGaming, "SetGamingProfile", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(value),
		}); err != nil {
			continue
		}
		time.Sleep(400 * time.Millisecond)
		if mode := activeMode(); mode != 0 {
			fmt.Printf("  profile %d -> ActiveMode=%d\n", value, mode)
		}
	}

	fmt.Println()
	fmt.Println("=== the generic action class ===")
	for query := uint64(0); query <= 0x30; query++ {
		if _, err := client.Call(hardware.ClassAction, "SetFunction", path, map[string]wmi.Value{
			"uiInput": wmi.Uint64(query),
		}); err != nil {
			continue
		}
		time.Sleep(400 * time.Millisecond)
		if mode := activeMode(); mode != 0 {
			fmt.Printf("  action %d -> ActiveMode=%d\n", query, mode)
		}
	}

	fmt.Println()
	fmt.Println("active mode at the end:", activeMode())
}

// activeMode reads ActiveMode from the lighting profile.
func activeMode() int {
	text, err := os.ReadFile(profilePath)
	if err != nil {
		return -1
	}
	match := regexp.MustCompile(`(?m)^ActiveMode=(\d+)`).FindStringSubmatch(string(text))
	if match == nil {
		return -1
	}
	mode, _ := strconv.Atoi(match[1])
	return mode
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

// keep exec referenced for future probes.
var _ = exec.Command
var _ = strings.TrimSpace
