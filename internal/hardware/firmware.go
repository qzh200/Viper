package hardware

// OpenFirmware opens the connection to the ACPI provider.
//
// The provider needs the rights of an administrator, so the connection is
// made only when a setting that needs it is used.
//
// 固件那一侧是**补充**：硬件层主要靠原厂程序，直接操控固件只在原厂程序给不了
// 的那几项上用得上（风扇表、超频能力、背光原始缓冲区、复位灯效、锁 Windows
// 键）。这一层要哪几项，由 internal/machine 的 Extras 接口说了算。
func OpenFirmware() (*Backend, error) {
	return NewBackend()
}
