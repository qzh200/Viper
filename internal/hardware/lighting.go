package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// The keyboard backlight hotkey identifier used by the action class.
const backlightHotkey = 132

// backlightTimeoutSeconds is the number of seconds after which the
// backlight switches off on its own.
const backlightTimeoutSeconds = 30

// Keyboard lighting related bits of the action class input word.
const (
	backlightReadBit  = 1
	backlightWriteBit = 2
	backlightFlagBit  = 0x80000
)

// The number of lighting zones the keyboard has.
const LightingZones = 4

// BacklightState describes the keyboard backlight.
type BacklightState struct {
	// Brightness is the backlight level, 0 to 100.
	Brightness int
	// Timeout reports whether the backlight switches off after 30 seconds.
	Timeout bool
}

// SetLighting applies a keyboard lighting configuration.
//
// The provider applies the effect, the speed and the direction to the whole
// keyboard at once, while the colour is written per zone. A global setup
// therefore writes the same colour to every zone.
//
// 速度、方向和亮度在这里先被限制到固件能接受的范围，和上层写进配置文件的
// 范围一样，这样写进固件的值和记下来的值是同一个值。
func (b *Backend) SetLighting(config RGBConfig) error {
	_, err := operate(b, func() (struct{}, error) {
		return struct{}{}, b.applyLighting(config)
	})
	return err
}

// applyLighting writes the lighting without holding the lock.
//
// 锁加在公开的那一层，内部的辅助方法不再加锁，所以一个操作里可以调用另一个
// 操作而不会自己等自己。
func (b *Backend) applyLighting(config RGBConfig) error {
	speed := clampUint(config.Speed, 1, 9)
	direction := clampUint(config.Direction, 0, 1)
	brightness := clampUint(config.Brightness, 0, 100)

	colours := config.Zones
	if !config.PerZone {
		for i := 1; i < LightingZones; i++ {
			colours[i] = colours[0]
		}
	}

	for zone := 0; zone < LightingZones; zone++ {
		if _, err := b.callGaming("SetGamingLEDColor", map[string]wmi.Value{
			"gmInput": wmi.Uint64(colorWord(colours[zone], uint64(zone+1))),
		}); err != nil {
			return fmt.Errorf("hardware: 设置灯光分区 %d 失败: %w", zone+1, err)
		}
	}

	behavior := (uint64(config.Effect) << 16) |
		(speed << 8) |
		direction

	if _, err := b.callGaming("SetGamingLEDBehavior", map[string]wmi.Value{
		"gmInput": wmi.Uint64(behavior),
	}); err != nil {
		return fmt.Errorf("hardware: 设置灯光行为失败: %w", err)
	}

	// The automatic switch off is a setting of its own, changing the
	// lighting must not change it.
	//
	// 读不到这个开关的状态时不去写动作类，这样开关保持用户原来的选择，
	// 而不是被改成关闭。
	timeout, err := b.backlightTimeout()
	if err != nil {
		return b.setBacklight(int(brightness), nil)
	}
	return b.setBacklight(int(brightness), &timeout)
}

// ResetLighting returns the keyboard to the default appearance.
//
// The default is a static white effect in every zone at full brightness,
// which is what the official application does with its reset button.
func (b *Backend) ResetLighting() error {
	white := RGBColor{R: 255, G: 255, B: 255}
	config := RGBConfig{
		Effect:     EffectStatic,
		Speed:      3,
		Direction:  0,
		Brightness: 100,
		PerZone:    true,
	}
	for i := range config.Zones {
		config.Zones[i] = white
	}
	_, err := operate(b, func() (struct{}, error) {
		return struct{}{}, b.applyLighting(config)
	})
	return err
}

// Lighting reads the keyboard lighting configuration.
//
// Only the brightness can be read back from the firmware. The provider
// reports neither the colour nor the effect of the zones, so those fields
// are reported as unsupported and the caller keeps its own record.
func (b *Backend) Lighting() (RGBConfig, error) {
	var config RGBConfig

	brightness, err := b.LightingBrightness()
	if err != nil {
		return config, err
	}
	config.Brightness = brightness

	config.ColourReadable = false
	config.EffectReadable = false
	return config, nil
}

// LightingBrightness reads the keyboard brightness.
//
// GetGamingRgbKb reports the level of the keyboard in its second byte. The
// provider clamps the level to the range from zero to one hundred, so a
// larger value in the firmware is an uninitialised state rather than a
// brighter keyboard.
func (b *Backend) LightingBrightness() (uint64, error) {
	outputs, err := b.callGaming("GetGamingRgbKb", map[string]wmi.Value{
		"gmInput": wmi.Uint32(1),
	})
	if err != nil {
		return 0, err
	}
	// 固件没有给出 gmOutput 时不能当成读到了零，那会让上层把"读不到"当成
	// "键盘是灭着的"。
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("hardware: 读取键盘亮度时固件没有给出 gmOutput")
	}
	raw, err := decodeUint(value)
	if err != nil {
		return 0, err
	}
	level := (raw >> 8) & 0xFF
	if level > 100 {
		level = 100
	}
	return level, nil
}

// colorWord builds the word that carries the colour of one zone.
//
// The word is blue, green and red from the top down and the zone number in
// the lowest byte. The zones are counted from one.
func colorWord(color RGBColor, zone uint64) uint64 {
	return (uint64(color.B) << 24) |
		(uint64(color.G) << 16) |
		(uint64(color.R) << 8) |
		zone
}

// NewBacklightBuffer builds the sixteen byte buffer of the keyboard
// backlight.
//
// The layout was read back from the provider with GetGamingKBBacklight:
// the level sits in byte two and in byte five, byte eight holds the flags
// of the backlight and byte nine has to be one, otherwise the provider
// drops the buffer and zeroes the level.
//
// The function is public, so that a stand for the machine builds the
// buffer the same way the machine does. A stand that builds its own
// buffer would answer in a different shape than the hardware and the
// tests of the layers above would not notice a wrong layout.
func NewBacklightBuffer(level int) []byte {
	buffer := make([]byte, 16)
	buffer[2] = byte(level)
	buffer[5] = byte(level)
	buffer[8] = backlightFlags
	buffer[9] = 1
	return buffer
}

// backlightBuffer builds the sixteen byte buffer of the keyboard backlight.
func backlightBuffer(level int) []byte {
	return NewBacklightBuffer(level)
}

// SetBacklight sets the keyboard backlight brightness.
//
// The provider expects the level in the generic action class and again in
// the Acer gaming class, because the two classes drive different parts of the
// keyboard controller.
func (b *Backend) SetBacklight(brightness int, timeout bool) error {
	_, err := operate(b, func() (struct{}, error) {
		return struct{}{}, b.setBacklight(brightness, &timeout)
	})
	return err
}

// setBacklight writes the backlight.
//
// timeout 为 nil 的时候不去写动作类，这样 30 秒自动熄灭的开关保持原来的
// 状态，不会被改写成关闭。
func (b *Backend) setBacklight(brightness int, timeout *bool) error {
	level := clampPercent(brightness)

	if timeout != nil {
		timeoutByte := uint64(0)
		if *timeout {
			timeoutByte = backlightTimeoutSeconds
		}

		actionInput := uint64(backlightWriteBit) |
			(uint64(backlightHotkey) << 8) |
			backlightFlagBit |
			(uint64(level) << 32) |
			(timeoutByte << 40)

		if _, err := b.callAction("SetFunction", map[string]wmi.Value{
			"uiInput": wmi.Uint64(actionInput),
		}); err != nil {
			return fmt.Errorf("hardware: 设置键盘背光失败: %w", err)
		}
	}

	// The Acer gaming class takes the level as a sixteen byte buffer.
	if _, err := b.callGaming("SetGamingKBBacklight", map[string]wmi.Value{
		"gmInput": wmi.Bytes(backlightBuffer(level)),
	}); err != nil {
		return fmt.Errorf("hardware: 设置键盘背光缓冲区失败: %w", err)
	}

	// 把缓冲区读回来核对亮度，固件没有接受这个亮度就说清楚是哪一个。
	if err := b.verifyBacklightBuffer(level); err != nil {
		return err
	}

	if _, err := b.callGaming("SetGamingRgbKb", map[string]wmi.Value{
		"gmInput": wmi.Uint64(1 | (uint64(level) << 8)),
	}); err != nil {
		return fmt.Errorf("hardware: 设置键盘背光亮度失败: %w", err)
	}

	return nil
}

// BacklightBuffer reads the raw buffer of the keyboard backlight.
//
// The buffer is sixteen bytes long. The level sits in byte two and in byte
// five, byte eight holds the flags of the backlight (three means that the
// backlight switches itself off after thirty seconds) and byte nine has to
// be one when the buffer is written.
//
// 固件把缓冲区读回来的时候第九个字节是零，所以核对亮度时只看第二个和第五个
// 字节。
func (b *Backend) BacklightBuffer() ([]byte, error) {
	outputs, err := b.callGaming("GetGamingKBBacklight", map[string]wmi.Value{
		"gmInput": wmi.Uint32(1),
	})
	if err != nil {
		return nil, err
	}

	value, ok := outputs["gmOutput"]
	if !ok {
		return nil, fmt.Errorf("hardware: GetGamingKBBacklight 没有返回 gmOutput")
	}

	buffer, err := decodeBytes(value)
	if err != nil {
		return nil, fmt.Errorf("hardware: 读回背光缓冲区失败: %w", err)
	}
	return buffer, nil
}

// verifyBacklightBuffer reads the backlight buffer back and compares the
// level with the one that was written.
//
// 读不回缓冲区就没有可以比较的值，这种情况下跳过核对。
func (b *Backend) verifyBacklightBuffer(level int) error {
	buffer, err := b.BacklightBuffer()
	if err != nil {
		return nil
	}
	if len(buffer) < 6 {
		return fmt.Errorf("hardware: 读回的背光缓冲区只有 %d 个字节，期望 16 个", len(buffer))
	}
	if int(buffer[2]) != level || int(buffer[5]) != level {
		return fmt.Errorf("hardware: 固件没有接受背光亮度 %d，缓冲区里第二个字节是 %d，第五个字节是 %d",
			level, buffer[2], buffer[5])
	}
	return nil
}

// backlightFlags is the value of the flags byte of the backlight buffer.
//
// The provider reports three for a keyboard that switches its backlight off
// after thirty seconds.
const backlightFlags = 3

// Backlight reads the keyboard backlight state.
//
// The level comes from the gaming class, the automatic switch off comes
// from the action class, which keeps it in the sixth byte of its answer.
func (b *Backend) Backlight() (BacklightState, error) {
	var state BacklightState

	brightness, err := b.LightingBrightness()
	if err != nil {
		return state, err
	}
	state.Brightness = int(brightness)

	timeout, err := b.backlightTimeout()
	if err != nil {
		return state, err
	}
	state.Timeout = timeout
	return state, nil
}

// backlightTimeout reads the automatic switch off of the backlight.
//
// The switch lives in the action class, its answer keeps the number of
// seconds in the sixth byte. The lowest byte of the answer is the status of
// the setting, a status that is not zero means that the action class did not
// answer this setting at all.
//
// 读不到这个开关时必须报错：把它当成"关闭"会把用户原来开着的自动熄灭改掉。
func (b *Backend) backlightTimeout() (bool, error) {
	query := uint64(backlightReadBit) |
		(uint64(backlightHotkey) << 8) |
		backlightFlagBit

	outputs, err := b.callAction("GetFunction", map[string]wmi.Value{
		"uiInput": wmi.Uint32(uint32(query)),
	})
	if err != nil {
		return false, err
	}
	raw, err := decodeUint(outputs["uiOutput"])
	if err != nil {
		return false, err
	}
	if raw&0xFF != 0 {
		return false, fmt.Errorf("hardware: 动作类没有回答背光的自动熄灭开关，状态字节是 %d", raw&0xFF)
	}
	return (raw>>40)&0xFF == backlightTimeoutSeconds, nil
}
