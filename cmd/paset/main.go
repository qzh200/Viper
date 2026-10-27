//go:build tools

// Command paset records the answers of the service to the requests that
// write a setting.
//
// Only the values that the machine already has are written, so nothing
// changes.
package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type request struct {
	packet uint32
	body   string
}

func main() {
	address := "127.0.0.1:46933"
	if len(os.Args) > 1 {
		address = os.Args[1]
	}

	requests := []request{
		{100, `{"Function":"OPERATING_MODE","Parameter":{"mode":1}}`},
		{100, `{"Function":"GPU_MODE","Parameter":{"mode":1}}`},
		{100, `{"Function":"STICKY_KEY","Parameter":{"status":0}}`},
		{100, `{"Function":"BACKLIGHT_30S","Parameter":{"status":1}}`},
		{100, `{"Function":"LCD_OVERDRIVE","Parameter":{"status":0}}`},
		{100, `{"Function":"BOOT_SOUND","Parameter":{"status":0}}`},
		{100, `{"Function":"WIN_KEY","Parameter":{"status":1}}`},
	}

	for _, item := range requests {
		answer, err := ask(address, item.packet, item.body)
		if err != nil {
			fmt.Printf("### %s -> %v\n", item.body, err)
			continue
		}
		fmt.Printf("### %s\n    %q\n", item.body, answer)
	}
}

// ask sends one request and reads the answer.
func ask(address string, packet uint32, body string) (string, error) {
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

var _ = strings.TrimSpace
