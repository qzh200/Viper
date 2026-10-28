//go:build !windows

package hardware

// getSystemFirmwareTable 在非 Windows 平台上的空实现。
func getSystemFirmwareTable(provider uint32, tableID uint32, buffer []byte) uint32 {
	return 0
}
