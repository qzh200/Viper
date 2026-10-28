//go:build windows

package hardware

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	spiGetStickyKeys = 0x003A
	spiSetStickyKeys = 0x003B
	spiUpdateIniFile = 0x0001
	spiSendChange    = 0x0002
)

const (
	stickyKeysOn        = 0x00000001
	stickyKeysAvailable = 0x00000002
)

const stickyKeysPath = `Control Panel\Accessibility\StickyKeys`

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	procSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
)

type stickyKeys struct {
	size  uint32
	flags uint32
}

// StickyKeys 查询 Windows 粘滞键当前是否处于开启状态。
func StickyKeys() (bool, error) {
	keys, err := readStickyKeys()
	if err != nil {
		return false, err
	}
	return keys.flags&stickyKeysOn != 0, nil
}

// SetStickyKeys 启用或禁用 Windows 粘滞键功能。
func SetStickyKeys(enabled bool) error {
	keys, err := readStickyKeys()
	if err != nil {
		return err
	}

	if enabled {
		keys.flags |= stickyKeysOn
	} else {
		keys.flags &^= stickyKeysOn
	}
	keys.flags |= stickyKeysAvailable

	r, _, callErr := procSystemParametersInfoW.Call(
		uintptr(spiSetStickyKeys),
		uintptr(keys.size),
		uintptr(unsafe.Pointer(&keys)),
		uintptr(spiUpdateIniFile|spiSendChange),
	)
	if r == 0 {
		return fmt.Errorf("hardware: the system refused to change the sticky keys: %w", callErr)
	}
	return nil
}

func readStickyKeys() (stickyKeys, error) {
	keys := stickyKeys{size: uint32(unsafe.Sizeof(stickyKeys{}))}

	r, _, _ := procSystemParametersInfoW.Call(
		uintptr(spiGetStickyKeys),
		uintptr(keys.size),
		uintptr(unsafe.Pointer(&keys)),
		0,
	)
	if r != 0 {
		return keys, nil
	}

	flags, err := registryStickyKeysFlags()
	if err != nil {
		return keys, fmt.Errorf("hardware: read the sticky keys: %w", err)
	}
	keys.flags = flags
	return keys, nil
}

func registryStickyKeysFlags() (uint32, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, stickyKeysPath, registry.QUERY_VALUE)
	if err != nil {
		return 0, fmt.Errorf("hardware: open the sticky keys setting: %w", err)
	}
	defer key.Close()

	val, _, err := key.GetStringValue("Flags")
	if err != nil {
		return 0, fmt.Errorf("hardware: read the sticky keys flags: %w", err)
	}

	var flags uint32
	if _, err := fmt.Sscanf(val, "%d", &flags); err != nil {
		return 0, fmt.Errorf("hardware: parse the sticky keys flags: %w", err)
	}
	return flags, nil
}
