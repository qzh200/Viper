//go:build tools

// Command gpucross compares the readings of the graphics card with the
// ones of the original software.
//
// The original service reads the card through the interface of the driver,
// Viper reads it through the management library. Both should tell the same
// story.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"viper/internal/nvidia"
)

func main() {
	cards, err := nvidia.Read()
	if err != nil {
		fmt.Println("nvidia:", err)
		os.Exit(1)
	}

	fmt.Println("=== Viper, management library ===")
	for i, card := range cards {
		fmt.Printf("  card %d: %s\n", i, card.Name)
		fmt.Printf("    usage        : %d%% (memory %d%%)\n", card.UsagePercent, card.MemoryUsagePercent)
		fmt.Printf("    graphics clock: %d MHz (max %d MHz)\n", card.GraphicsClockMHz, card.MaxGraphicsClockMHz)
		fmt.Printf("    memory clock : %d MHz\n", card.MemoryClockMHz)
		fmt.Printf("    memory       : %d of %d bytes\n", card.MemoryUsedBytes, card.MemoryTotalBytes)
		fmt.Printf("    performance  : %d\n", card.PState)
	}

	version, err := nvidia.DriverVersion()
	if err == nil {
		fmt.Println("  driver        :", version)
	}

	fmt.Println("\n=== original service, interface of the driver ===")
	answer, err := ask("127.0.0.1:46933", 10, map[string]any{})
	if err != nil {
		fmt.Println("  the service did not answer:", err)
		return
	}
	printAnswer(answer)
}

// printAnswer shows the parts of an answer that carry the graphics card.
func printAnswer(answer map[string]any) {
	data, ok := answer["data"]
	if !ok {
		fmt.Printf("  %v\n", answer)
		return
	}
	document, err := json.MarshalIndent(data, "  ", "  ")
	if err != nil {
		fmt.Printf("  %v\n", data)
		return
	}
	text := string(document)
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "gpu") || strings.Contains(lower, "frequency") ||
			strings.Contains(lower, "usage") || strings.Contains(lower, "idle") ||
			strings.Contains(lower, "clock") {
			fmt.Println(" ", strings.TrimSpace(line))
		}
	}
}

// ask sends one request to the service of the original software.
func ask(address string, id uint32, payload map[string]any) (map[string]any, error) {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

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
	n, err := conn.Read(buffer)
	if n == 0 {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no answer")
	}

	text := string(buffer[:n])
	start := strings.Index(text, "{")
	if start < 0 {
		return nil, fmt.Errorf("unexpected answer %q", text)
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
					return nil, err
				}
				return answer, nil
			}
		}
	}
	return nil, io.ErrUnexpectedEOF
}
