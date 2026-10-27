//go:build tools

// Command paset2 sends one request that writes a setting.
//
// The request is given as a whole, so that the payload can be copied from
// the traffic of the official application.
package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	address := "127.0.0.1:46934"
	if len(os.Args) > 2 {
		address = os.Args[1]
	}
	packet := uint32(100)
	if len(os.Args) > 3 {
		value, err := strconv.ParseUint(os.Args[2], 10, 32)
		if err == nil {
			packet = uint32(value)
		}
	}
	body := ""
	if len(os.Args) > 2 {
		argument := os.Args[len(os.Args)-1]
		if strings.HasSuffix(argument, ".json") {
			// The body is read from a file, because the shell mangles the
			// quotes of a JSON document.
			data, err := os.ReadFile(argument)
			if err != nil {
				fmt.Println("read:", err)
				return
			}
			body = string(data)
		} else {
			body = argument
		}
	}
	if body == "" {
		fmt.Println("usage: paset2 <address> <json file> [packet]")
		return
	}

	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer conn.Close()

	frame := make([]byte, 0, len(body)+8)
	frame = append(frame, 'A', 'C', 'E', 'R')
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], packet)
	frame = append(frame, header[:]...)
	frame = append(frame, body...)

	if _, err := conn.Write(frame); err != nil {
		fmt.Println("write:", err)
		return
	}

	conn.SetReadDeadline(time.Now().Add(4 * time.Second))
	buffer := make([]byte, 65536)
	n, err := conn.Read(buffer)
	if n == 0 {
		if err != nil {
			fmt.Println("read:", err)
			return
		}
		fmt.Println("no answer")
		return
	}
	fmt.Printf("%q\n", string(buffer[:n]))
}
