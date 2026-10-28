//go:build windows

package hardware

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemFirmwareTable = kernel32.NewProc("GetSystemFirmwareTable")
)

// getSystemFirmwareTable 封装 Windows API GetSystemFirmwareTable 调用以检索固件表。
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
