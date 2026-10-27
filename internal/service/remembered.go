package service

import (
	"errors"
	"fmt"
)

// errNoBackend is the error that the service gives when a setting needs the
// gaming provider and the provider is not there.
//
// The provider needs the rights of an administrator, the service runs
// without them and reports the settings that it can.
var errNoBackend = errors.New("service: 游戏硬件提供者不可用，需要管理员权限")

// The defaults of the settings that the firmware cannot give back.
//
// They are the values that the original service reports on a machine that
// has never changed them.
const (
	// defaultBatteryBoost is the state of the battery boost.
	defaultBatteryBoost = 1
	// defaultSoundFunction is the chip that carries the sound profiles.
	defaultSoundFunction = "DTS"
	// defaultSoundMode is the profile of the sound chip.
	defaultSoundMode = 0
	// defaultBootLogo is the picture of the boot logo, an empty text
	// means that the machine uses the picture of the vendor.
	defaultBootLogo = ""
)

// rememberedSettings holds the settings that the service remembers itself.
//
// The firmware of the machine cannot be asked for them: the battery boost,
// the sound profile and the boot logo have no interface in the gaming
// provider. The values that the service writes are therefore **its own
// record**, they are never read back from the firmware.
type rememberedSettings struct {
	// batteryBoost tells whether the machine runs faster on the battery.
	batteryBoost int
	// batteryBoostSet tells whether the battery boost was written.
	batteryBoostSet bool

	// soundFunction is the chip that carries the sound profiles.
	soundFunction string
	// soundMode is the profile of the sound chip.
	soundMode int
	// soundSet tells whether the sound profile was written.
	soundSet bool

	// bootLogo is the path of the picture of the boot logo.
	bootLogo string
	// bootLogoSet tells whether the boot logo was written.
	bootLogoSet bool
}

// batteryBoostState returns the battery boost that the service remembers.
//
// The value is the service's own record, the firmware has no interface for
// it. A machine that never changed it reports the value of the original
// service.
func (p *Provider) batteryBoostState() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.remembered.batteryBoostSet {
		return defaultBatteryBoost
	}
	return p.remembered.batteryBoost
}

// rememberBatteryBoost stores the battery boost that was written.
func (p *Provider) rememberBatteryBoost(status int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.remembered.batteryBoost = status
	p.remembered.batteryBoostSet = true
}

// soundModeState returns the sound profile that the service remembers.
//
// The value is the service's own record, the sound chip of the vendor is
// not part of the gaming provider. A machine that never changed it
// reports the profile of the original service.
func (p *Provider) soundModeState() (string, int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.remembered.soundSet {
		return defaultSoundFunction, defaultSoundMode
	}
	return p.remembered.soundFunction, p.remembered.soundMode
}

// rememberSoundMode stores the sound profile that was written.
func (p *Provider) rememberSoundMode(function string, mode int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.remembered.soundFunction = function
	p.remembered.soundMode = mode
	p.remembered.soundSet = true
}

// bootLogoState returns the boot logo that the service remembers.
//
// The value is the service's own record, the firmware of this machine does
// not support a boot logo at all. A machine that never changed it reports
// an empty text, the same as the original service.
func (p *Provider) bootLogoState() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.remembered.bootLogoSet {
		return defaultBootLogo
	}
	return p.remembered.bootLogo
}

// rememberBootLogo stores the boot logo that was written.
func (p *Provider) rememberBootLogo(source string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.remembered.bootLogo = source
	p.remembered.bootLogoSet = true
}

// applyBatteryBoost writes the battery boost.
//
// The firmware of the machine has no interface for it, so the service
// remembers the value and reports it later. It is the service's own
// record, not a value that was read back from the firmware.
func (p *Provider) applyBatteryBoost(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return fmt.Errorf("service: 电池加速的数值无效: %w", err)
	}

	p.rememberBatteryBoost(boolStatus(value != 0))
	return nil
}

// applySoundMode writes the sound profile.
//
// The sound chip of the vendor is not part of the gaming provider, so the
// service remembers the profile and reports it later. It is the service's
// own record, not a value that was read back from the firmware.
func (p *Provider) applySoundMode(parameter any) error {
	values, ok := parameter.(map[string]any)
	if !ok {
		return fmt.Errorf("service: 声音模式需要一组数值")
	}

	function := defaultSoundFunction
	if text, ok := values["function"].(string); ok && text != "" {
		function = text
	}
	mode := defaultSoundMode
	if value, err := number(values, "mode"); err == nil {
		mode = int(value)
	}

	p.rememberSoundMode(function, mode)
	return nil
}

// applyBootLogo writes the boot logo.
//
// The firmware of this machine does not support a boot logo, so the service
// remembers the picture and reports it later. It is the service's own
// record, not a value that was read back from the firmware.
func (p *Provider) applyBootLogo(parameter any) error {
	values, ok := parameter.(map[string]any)
	if !ok {
		return fmt.Errorf("service: 启动标志需要一组数值")
	}

	source, _ := values["imageSource"].(string)
	p.rememberBootLogo(source)
	return nil
}
