//go:build tools

// Command pacapture records the answers of the PredatorSense service.
//
// The official application talks to the service over a local socket. The
// tool sends every request the application is known to send and writes the
// raw answer to a file, so that the framing and the shape of the answers
// can be looked at byte by byte.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

// The packet identifiers of the protocol.
const (
	packetInitialization     = 0
	packetGetMonitorData     = 10
	packetGetUpdatedData     = 20
	packetGetConnectedDevice = 30
	packetGetLightingEffect  = 40
	packetSetDeviceData      = 100
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
		{packetInitialization, `{"Function":"VERSION"}`},
		{packetInitialization, `{"Function":"SUPPORT_OPERATING_MODE_CAPABILITY"}`},
		{packetInitialization, `{"Function":"SUPPORT_AUDIO_MODE"}`},
		{packetInitialization, `{"Function":"SUPPORT_GPU_MODE"}`},
		{packetInitialization, `{"Function":"SUPPORT_GPU_MODE_CAPABILITY"}`},
		{packetInitialization, `{"Function":"SUPPORT_LCD_OVERDRIVE"}`},
		{packetInitialization, `{"Function":"SUPPORT_CUSTOM_BOOT_LOGO"}`},
		{packetInitialization, `{"Function":"SUPPORT_KEYBOARD_INFO"}`},
		{packetInitialization, `{"Function":"SUPPORT_FAN_CONTROL"}`},
		{packetInitialization, `{"Function":"SUPPORT_CPU_INFO"}`},
		{packetInitialization, `{"Function":"SUPPORT_GPU_INFO"}`},
		{packetInitialization, `{"Function":"SUPPORT_OVERCLOCKING_MODE_CAPABILITY"}`},
		{packetGetMonitorData, `{}`},
		{packetGetUpdatedData, `{"Function":"OPERATING_MODE"}`},
		{packetGetUpdatedData, `{"Function":"GPU_MODE"}`},
		{packetGetUpdatedData, `{"Function":"SOUND_MODE"}`},
		{packetGetUpdatedData, `{"Function":"STICKY_KEY"}`},
		{packetGetUpdatedData, `{"Function":"BACKLIGHT_30S"}`},
		{packetGetUpdatedData, `{"Function":"WIN_KEY"}`},
		{packetGetUpdatedData, `{"Function":"LCD_OVERDRIVE"}`},
		{packetGetUpdatedData, `{"Function":"BOOT_SOUND"}`},
		{packetGetUpdatedData, `{"Function":"ADAPTOR_STATUS"}`},
		{packetGetUpdatedData, `{"Function":"BATTERY_BOOST"}`},
		{packetGetUpdatedData, `{"Function":"FAN_CONTROL"}`},
		{packetGetUpdatedData, `{"Function":"LIGHTING"}`},
		{packetGetUpdatedData, `{"Function":"CUSTOM_BOOT_LOGO"}`},
		{packetGetConnectedDevice, `{}`},
		{packetGetLightingEffect, `{}`},
	}

	file, err := os.Create("E:/PredatorSenseGo/_scratch/pacapture.txt")
	if err != nil {
		fmt.Println("capture:", err)
		return
	}
	defer file.Close()

	for _, item := range requests {
		answer, err := ask(address, item.packet, item.body)
		if err != nil {
			fmt.Printf("packet %d %s -> %v\n", item.packet, item.body, err)
			continue
		}
		fmt.Fprintf(file, "### packet %d %s\n", item.packet, item.body)
		fmt.Fprintf(file, "raw : %q\n", answer)
		fmt.Fprintf(file, "len : %d\n", len(answer))
		fmt.Fprintf(file, "hex : %x\n\n", answer)
		fmt.Printf("packet %d %-52s -> %d bytes\n", item.packet, item.body, len(answer))
	}
}

// ask sends one request and reads the raw answer.
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

// unused keeps the encoding of json in the imports, the answers are
// checked by eye.
var _ = json.Marshal
