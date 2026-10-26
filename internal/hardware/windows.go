//go:build windows

package hardware

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// The sticky keys live in the accessibility settings of the current user.
//
// The original application reaches them through the same system call, the
// accessibility agent of Windows watches the setting and applies it without
// a restart.
const (
	// spiGetStickyKeys reads the sticky key configuration.
	spiGetStickyKeys = 0x003A
	// spiSetStickyKeys writes the sticky key configuration.
	spiSetStickyKeys = 0x003B
	// spiUpdateIniFile stores the new value in the profile.
	spiUpdateIniFile = 0x0001
	// spiSendChange tells the running system about the new value.
	spiSendChange = 0x0002
)

// The bits of the sticky key configuration.
const (
	// stickyKeysOn marks the sticky keys as switched on.
	stickyKeysOn = 0x00000001
	// stickyKeysAvailable marks the feature as present on this machine.
	stickyKeysAvailable = 0x00000002
)

// stickyKeysPath is where Windows keeps the state of the sticky keys.
const stickyKeysPath = `Control Panel\Accessibility\StickyKeys`

// user32 holds the system calls of the user interface library.
var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	procSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
)

// stickyKeys is the structure the system exchanges for the sticky keys.
//
// It carries the size of the structure and the flag word, the lowest bit of
// the word carries the state of the switch.
type stickyKeys struct {
	size  uint32
	flags uint32
}

// StickyKeys reports whether the sticky keys are switched on.
func StickyKeys() (bool, error) {
	keys, err := readStickyKeys()
	if err != nil {
		return false, err
	}
	return keys.flags&stickyKeysOn != 0, nil
}

// SetStickyKeys switches the sticky keys on or off.
//
// The change is written to the profile of the current user and announced to
// the running system, so the keyboard reacts right away.
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

// readStickyKeys reads the sticky key configuration.
//
// The system call is the accurate source, the registry is the fallback for
// the rare case where the call is refused.
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

// registryStickyKeysFlags reads the sticky key flags from the profile.
func registryStickyKeysFlags() (uint32, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, stickyKeysPath, registry.QUERY_VALUE)
	if err != nil {
		return 0, fmt.Errorf("hardware: open the sticky keys setting: %w", err)
	}
	defer key.Close()

	value, _, err := key.GetIntegerValue("Flags")
	if err != nil {
		return 0, fmt.Errorf("hardware: read the sticky keys setting: %w", err)
	}
	return uint32(value), nil
}
