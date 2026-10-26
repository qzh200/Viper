//go:build tools

// Command activemode finds how the active lighting mode is switched.
//
// Writing the parameters of a mode does not activate it, so the active
// mode is read from the profile of the Acer lighting service after every
// candidate call.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

// profilePath is the profile of the Acer lighting service.
const profilePath = `C:\ProgramData\OEM\AcerLightingService\LightingProfile\LightingProfile.ini`

func main() {
	client, path := connect()

	fmt.Println("=== profile before ===")
	printProfile()

	fmt.Println()
	fmt.Println("=== try each effect value ===")
	for effect := uint64(0); effect <= 8; effect++ {
		if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(effect<<16 | 5<<8 | 0),
		}); err != nil {
			fmt.Printf("  effect %d -> %v\n", effect, err)
			continue
		}
		time.Sleep(700 * time.Millisecond)
		mode, names := activeProfile()
		fmt.Printf("  effect %d -> ActiveMode=%d modes=%v\n", effect, mode, names)
	}

	fmt.Println()
	fmt.Println("=== try SetGamingLED with a twelve byte structure ===")
	for _, data := range [][]byte{
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00},
		{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00},
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	} {
		if _, err := client.Call(hardware.ClassGaming, "SetGamingLED", path, map[string]wmi.Value{
			"gmInput": wmi.Bytes(data),
		}); err != nil {
			fmt.Printf("  %v -> %v\n", data, err)
			continue
		}
		time.Sleep(700 * time.Millisecond)
		mode, names := activeProfile()
		fmt.Printf("  %v -> ActiveMode=%d modes=%v\n", data, mode, names)
	}

	fmt.Println()
	fmt.Println("=== profile after ===")
	printProfile()
}

// activeProfile reads the active mode and the mode names.
func activeProfile() (int, []string) {
	text, err := os.ReadFile(profilePath)
	if err != nil {
		return -1, nil
	}
	content := string(text)

	mode := -1
	if match := regexp.MustCompile(`(?m)^ActiveMode=(\d+)`).FindStringSubmatch(content); match != nil {
		mode, _ = strconv.Atoi(match[1])
	}

	var names []string
	for _, match := range regexp.MustCompile(`(?m)^Mode\d+=(.+)$`).FindAllStringSubmatch(content, -1) {
		names = append(names, strings.TrimSpace(match[1]))
	}
	return mode, names
}

// printProfile shows the active mode and the mode table.
func printProfile() {
	text, err := os.ReadFile(profilePath)
	if err != nil {
		fmt.Println("  cannot read:", err)
		return
	}
	content := string(text)
	for _, line := range strings.Split(content, "\n") {
		text := strings.TrimSpace(line)
		if strings.HasPrefix(text, "ActiveMode") || strings.HasPrefix(text, "Mode") ||
			strings.HasPrefix(text, "value=") || strings.HasPrefix(text, "color_mode=") ||
			strings.HasPrefix(text, "brightness=") {
			fmt.Println("  ", text)
		}
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
