package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// The number of lighting zones the keyboard has.
const LightingZones = 4

// backlightFlags is the flag byte of the backlight buffer.
//
// 三表示背光三十秒后自己熄灭。这个字节是背光开关自己的状态，写亮度的时候
// 要原样带回去，别把它改掉。
const backlightFlags = 3

// BacklightState describes the keyboard backlight.
type BacklightState struct {
	// Brightness is the backlight level, 0 to 100.
	Brightness int
	// Timeout reports whether the backlight switches off after 30 seconds.
	Timeout bool
}

// applyLighting writes the lighting without holding the lock.
//
// 锁加在公开的那一层，内部的辅助方法不再加锁，所以一个操作里可以调用另一个
// 操作而不会自己等自己。
//
// 它写三样东西：四个分区的颜色、灯效的行为（效果、速度、方向）和背光的
// 亮度。原厂程序能写这些，而且写得更完整（它还能读回分区颜色），所以这一路
// 只在原厂程序给不了的"复位灯效"上用。
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

	// 亮度写在背光缓冲区里，Acer 游戏类与动作类各管一半。三十秒自动熄灭的
	// 开关在这里不动——读不到它的状态，写它可能改掉用户的选择。
	if _, err := b.callGaming("SetGamingKBBacklight", map[string]wmi.Value{
		"gmInput": wmi.Bytes(NewBacklightBuffer(int(brightness))),
	}); err != nil {
		return fmt.Errorf("hardware: 设置键盘背光缓冲区失败: %w", err)
	}

	if _, err := b.callGaming("SetGamingRgbKb", map[string]wmi.Value{
		"gmInput": wmi.Uint64(1 | (uint64(brightness) << 8)),
	}); err != nil {
		return fmt.Errorf("hardware: 设置键盘背光亮度失败: %w", err)
	}

	return nil
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
//
// 固件读回来的时候第九个字节是零，所以核对亮度时只看第二个和第五个字节。
func NewBacklightBuffer(level int) []byte {
	buffer := make([]byte, 16)
	buffer[2] = byte(level)
	buffer[5] = byte(level)
	buffer[8] = backlightFlags
	buffer[9] = 1
	return buffer
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
		return nil, fmt.Errorf("hardware: 读取键盘背光缓冲区时固件没有给出 gmOutput")
	}
	buffer, err := decodeBytes(value)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}
