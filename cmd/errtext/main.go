//go:build windows

// Command errtext decodes WMI status codes into their message text.
package main

import (
	"fmt"

	"viper/internal/com"
)

func main() {
	for _, hr := range []uint32{
		0x80041001, 0x80041002, 0x80041003, 0x80041004,
		0x80041008, 0x8004100D, 0x80041017, 0x80041018,
		0x8004101A, 0x8004101D, 0x8004102A,
		0x800706F4, 0x80004002, 0x800401F0, 0x800401F1,
	} {
		fmt.Printf("0x%08X -> %s\n", hr, com.FormatHRESULT(hr))
	}
}
