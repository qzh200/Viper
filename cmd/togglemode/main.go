//go:build tools

// Command togglemode switches the active lighting mode through the
// profile of the Acer lighting service.
//
// The service owns the keyboard, so the profile is the place where the
// active mode lives.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const profilePath = `C:\ProgramData\OEM\AcerLightingService\LightingProfile\LightingProfile.ini`
const logPath = `C:\ProgramData\OEM\AcerLightingService\Log\SYSTEM_AcerOpenRGB_20261026.log`

func main() {
	original, err := os.ReadFile(profilePath)
	if err != nil {
		fmt.Println("cannot read the profile:", err)
		os.Exit(1)
	}
	fmt.Println("ActiveMode before:", activeMode(string(original)))

	for _, mode := range []int{3, 4, 0} {
		fmt.Printf("\n=== switch ActiveMode to %d ===\n", mode)
		writeActiveMode(mode)
		time.Sleep(2500 * time.Millisecond)

		after, err := os.ReadFile(profilePath)
		if err != nil {
			fmt.Println("  cannot read back:", err)
			continue
		}
		fmt.Println("  ActiveMode now:", activeMode(string(after)))
		fmt.Println("  profile changed by the service:", string(after) != string(original))
		tailLog()
	}

	fmt.Println()
	fmt.Println("=== restore the original profile ===")
	if err := os.WriteFile(profilePath, original, 0o644); err != nil {
		fmt.Println("  failed:", err)
	}
	after, _ := os.ReadFile(profilePath)
	fmt.Println("  ActiveMode now:", activeMode(string(after)))
}

// writeActiveMode replaces ActiveMode in the profile.
func writeActiveMode(mode int) {
	text, err := os.ReadFile(profilePath)
	if err != nil {
		fmt.Println("  cannot read:", err)
		return
	}
	updated := regexp.MustCompile(`(?m)^ActiveMode=\d+$`).
		ReplaceAllString(string(text), "ActiveMode="+strconv.Itoa(mode))
	if err := os.WriteFile(profilePath, []byte(updated), 0o644); err != nil {
		fmt.Println("  cannot write:", err)
	}
}

// activeMode reads ActiveMode from a profile.
func activeMode(text string) int {
	match := regexp.MustCompile(`(?m)^ActiveMode=(\d+)`).FindStringSubmatch(text)
	if match == nil {
		return -1
	}
	mode, _ := strconv.Atoi(match[1])
	return mode
}

// tailLog prints the last lines of the service log.
func tailLog() {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-Command",
		"Get-Content '"+logPath+"' -Tail 6").CombinedOutput()
	if err != nil {
		fmt.Println("  cannot read the log:", err)
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) != "" {
			fmt.Println("  ", strings.TrimSpace(line))
		}
	}
}
