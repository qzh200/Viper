//go:build tools

// Command palight talks to the lighting part of the PredatorSense service.
//
// The service owns the keyboard lighting and reports the full state of it,
// the effect, the speed, the direction, the brightness and the colour of
// every zone. That makes it possible to check whether a change written
// through the WMI provider really reaches the keyboard.
//
// The frame of a request is the magic word, the packet identifier as a
// little endian word and the payload as a JSON document.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const (
	// The address of the service.
	address = "127.0.0.1:46933"

	// The packet identifiers of the protocol.
	packetGetUpdatedData = 20
	packetSetDeviceData  = 100
)

func main() {
	command := "get"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer conn.Close()

	switch command {
	case "get":
		answer, err := ask(conn, packetGetUpdatedData, map[string]any{
			"Function": "LIGHTING",
		})
		if err != nil {
			fmt.Println("request:", err)
			return
		}
		printAnswer(answer)
	case "set":
		if len(os.Args) < 3 {
			fmt.Println("usage: palight set <json>")
			return
		}
		// The parameter can come from a file, because the quotes of a
		// JSON text do not survive the shell on every system.
		text := os.Args[2]
		if strings.HasPrefix(text, "@") {
			data, err := os.ReadFile(text[1:])
			if err != nil {
				fmt.Println("the parameter file cannot be read:", err)
				return
			}
			text = string(data)
		}
		var parameter any
		if err := json.Unmarshal([]byte(text), &parameter); err != nil {
			fmt.Println("the parameter is not valid json:", err)
			return
		}
		answer, err := ask(conn, packetSetDeviceData, map[string]any{
			"Function":  "LIGHTING",
			"Parameter": parameter,
		})
		if err != nil {
			fmt.Println("request:", err)
			return
		}
		printAnswer(answer)
	default:
		fmt.Println("usage: palight get | set <json>")
	}
}

// printAnswer shows the answer of the service.
func printAnswer(answer map[string]any) {
	data, err := json.MarshalIndent(answer, "", "  ")
	if err != nil {
		fmt.Println(answer)
		return
	}
	fmt.Println(string(data))
}

// ask sends one request and reads the answer.
func ask(conn net.Conn, id uint32, payload map[string]any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	frame := make([]byte, 0, len(body)+8)
	frame = append(frame, 'A', 'C', 'E', 'R')
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], id)
	frame = append(frame, header[:]...)
	frame = append(frame, body...)

	if _, err := conn.Write(frame); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 65536)
	pending := ""
	for {
		n, err := conn.Read(buffer)
		if n > 0 {
			pending += string(buffer[:n])
			if answer, ok := extractAnswer(pending); ok {
				return answer, nil
			}
		}
		if err != nil {
			if n == 0 {
				return nil, fmt.Errorf("the service did not answer: %w", err)
			}
			return nil, fmt.Errorf("the service did not finish the answer: %w", err)
		}
	}
}

// extractAnswer reads the first complete JSON document of a text.
func extractAnswer(text string) (map[string]any, bool) {
	start := strings.Index(text, "{")
	if start < 0 {
		return nil, false
	}
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				var answer map[string]any
				if err := json.Unmarshal([]byte(text[start:i+1]), &answer); err != nil {
					return nil, false
				}
				return answer, true
			}
		}
	}
	return nil, false
}
