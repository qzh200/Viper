//go:build !windows

package hardware

// getSystemFirmwareTable returns the size of the table when buffer is nil,
// and fills the buffer otherwise.
//
// 固件表只有 Windows 能读，别的系统上这个函数什么都不返回，读表的地方会
// 因此报"固件表不可用"。
func getSystemFirmwareTable(provider uint32, tableID uint32, buffer []byte) uint32 {
	return 0
}
