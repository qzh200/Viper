//go:build tools

// Command padiff compares the answers of the original service with the
// answers of the service of Viper.
//
// Both services are asked the same question and the answers are compared,
// the numbers that change on their own are listed apart from the ones that
// really differ.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"
)

type request struct {
	packet uint32
	body   string
}

// live are the numbers that change from moment to moment.
var live = map[string]bool{
	"CPU_CORE_AMOUNT":    true,
	"CPU_CORE_FREQUENCY": true,
	"CPU_FANSPEED":       true,
	"CPU_FREQUENCY":      true,
	"CPU_MAX_FREQUENCY":  true,
	"CPU_TEMPERATURE":    true,
	"CPU_USAGE":          true,
	"ETHERNET_DOWNLOAD":  true,
	"ETHERNET_UPLOAD":    true,
	"GPU1_FANSPEED":      true,
	"GPU1_FREQUENCY":     true,
	"GPU1_IDLE_STATUS":   true,
	"GPU1_MAX_FREQUENCY": true,
	"GPU1_TEMPERATURE":   true,
	"GPU1_USAGE":         true,
	"RAM_USAGE":          true,
	"SYS1_FANSPEED":      true,
	"SYS1_TEMPERATURE":   true,
	"WIFI_DOWNLOAD":      true,
	"WIFI_UPLOAD":        true,
	"brightness":         true,
	"color":              true,
	"direction":          true,
	"duration":           true,
	"speed":              true,
	"LEDs":               true,
	"effect":             true,
}

func main() {
	original := "127.0.0.1:46933"
	viper := "127.0.0.1:46934"
	if len(os.Args) > 2 {
		original = os.Args[1]
		viper = os.Args[2]
	}

	requests := []request{
		{0, `{"Function":"VERSION"}`},
		{0, `{"Function":"SUPPORT_OPERATING_MODE_CAPABILITY"}`},
		{0, `{"Function":"SUPPORT_AUDIO_MODE"}`},
		{0, `{"Function":"SUPPORT_GPU_MODE"}`},
		{0, `{"Function":"SUPPORT_GPU_MODE_CAPABILITY"}`},
		{0, `{"Function":"SUPPORT_LCD_OVERDRIVE"}`},
		{0, `{"Function":"SUPPORT_CUSTOM_BOOT_LOGO"}`},
		{0, `{"Function":"SUPPORT_KEYBOARD_INFO"}`},
		{0, `{"Function":"SUPPORT_FAN_CONTROL"}`},
		{0, `{"Function":"SUPPORT_CPU_INFO"}`},
		{0, `{"Function":"SUPPORT_GPU_INFO"}`},
		{10, `{}`},
		{20, `{"Function":"OPERATING_MODE"}`},
		{20, `{"Function":"GPU_MODE"}`},
		{20, `{"Function":"SOUND_MODE"}`},
		{20, `{"Function":"STICKY_KEY"}`},
		{20, `{"Function":"BACKLIGHT_30S"}`},
		{20, `{"Function":"WIN_KEY"}`},
		{20, `{"Function":"LCD_OVERDRIVE"}`},
		{20, `{"Function":"BOOT_SOUND"}`},
		{20, `{"Function":"ADAPTOR_STATUS"}`},
		{20, `{"Function":"BATTERY_BOOST"}`},
		{20, `{"Function":"FAN_CONTROL"}`},
		{20, `{"Function":"LIGHTING"}`},
		{20, `{"Function":"CUSTOM_BOOT_LOGO"}`},
		{30, `{}`},
		{40, `{}`},
	}

	differences := 0
	identical := 0
	for _, item := range requests {
		first, firstText, err := ask(original, item.packet, item.body)
		if err != nil {
			fmt.Printf("### %s -> the original service: %v\n", item.body, err)
			continue
		}
		second, secondText, err := ask(viper, item.packet, item.body)
		if err != nil {
			fmt.Printf("### %s -> viper: %v\n", item.body, err)
			differences++
			continue
		}

		paths := compare(first, second, "")
		if len(paths) == 0 {
			mark := "same"
			if firstText == secondText {
				identical++
				mark = "same, byte for byte"
			}
			fmt.Printf("### %-52s %s\n", item.body, mark)
			continue
		}

		fmt.Printf("### %s\n", item.body)
		for _, path := range paths {
			mark := "  "
			if !live[lastPart(path)] {
				mark = "!!"
				differences++
			}
			fmt.Printf("   %s %s\n", mark, path)
		}
	}

	fmt.Printf("\nidentical answers: %d, differences to look at: %d\n", identical, differences)
}

// lastPart gives the name of the last step of a path.
func lastPart(path string) string {
	if index := strings.LastIndex(path, "."); index >= 0 {
		return path[index+1:]
	}
	return path
}

// compare lists the paths where the two answers differ.
func compare(first, second map[string]any, prefix string) []string {
	var out []string

	keys := map[string]bool{}
	for key := range first {
		keys[key] = true
	}
	for key := range second {
		keys[key] = true
	}
	names := make([]string, 0, len(keys))
	for key := range keys {
		names = append(names, key)
	}
	sort.Strings(names)

	for _, key := range names {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		left, okLeft := first[key]
		right, okRight := second[key]
		switch {
		case !okLeft:
			out = append(out, path+"  only in viper")
		case !okRight:
			out = append(out, path+"  only in the original")
		default:
			leftMap, leftIsMap := left.(map[string]any)
			rightMap, rightIsMap := right.(map[string]any)
			if leftIsMap && rightIsMap {
				out = append(out, compare(leftMap, rightMap, path)...)
				continue
			}
			if !equal(left, right) {
				out = append(out, fmt.Sprintf("%s  original=%v viper=%v", path, left, right))
			}
		}
	}
	return out
}

// equal compares two values of a document.
func equal(left, right any) bool {
	leftText, leftOK := left.(string)
	rightText, rightOK := right.(string)
	if leftOK && rightOK {
		return leftText == rightText
	}
	leftNumber, leftIsNumber := left.(float64)
	rightNumber, rightIsNumber := right.(float64)
	if leftIsNumber && rightIsNumber {
		return leftNumber == rightNumber
	}
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
}

// ask sends one request and reads the answer.
func ask(address string, packet uint32, body string) (map[string]any, string, error) {
	text, err := askRaw(address, packet, body)
	if err != nil {
		return nil, "", err
	}
	var answer map[string]any
	if err := json.Unmarshal([]byte(text), &answer); err != nil {
		return nil, "", err
	}
	return answer, text, nil
}

// askRaw sends one request and reads the answer as text.
func askRaw(address string, packet uint32, body string) (string, error) {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	frame := make([]byte, 0, len(body)+8)
	frame = append(frame, 'A', 'C', 'E', 'R')
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], packet)
	frame = append(frame, header[:]...)
	frame = append(frame, body...)

	if _, err := conn.Write(frame); err != nil {
		return "", err
	}

	conn.SetReadDeadline(time.Now().Add(4 * time.Second))
	buffer := make([]byte, 65536)
	out := make([]byte, 0, 65536)
	depth := 0
	started := false
	for {
		n, err := conn.Read(buffer)
		if n > 0 {
			out = append(out, buffer[:n]...)
			for _, b := range buffer[:n] {
				switch b {
				case '{':
					depth++
					started = true
				case '}':
					depth--
				}
			}
			if started && depth == 0 {
				return string(out), nil
			}
		}
		if err != nil {
			if len(out) > 0 {
				return string(out), nil
			}
			return "", err
		}
	}
}
