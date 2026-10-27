//go:build windows

package hardware

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// The firmware tables are read with the call of the kernel, it is not
// wrapped by golang.org/x/sys/windows.
//
// The call only reads the tables the firmware already exposes, it never
// writes anything.
var (
	kernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemFirmwareTable = kernel32.NewProc("GetSystemFirmwareTable")
)

// getSystemFirmwareTable returns the size of the table when buffer is nil,
// and fills the buffer otherwise.
func getSystemFirmwareTable(provider uint32, tableID uint32, buffer []byte) uint32 {
	var pointer uintptr
	if len(buffer) > 0 {
		pointer = uintptr(unsafe.Pointer(&buffer[0]))
	}
	r, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(provider),
		uintptr(tableID),
		pointer,
		uintptr(len(buffer)),
	)
	return uint32(r)
}
