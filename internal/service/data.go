package service

import (
	"fmt"
	"reflect"
	"sync"

	"viper/internal/config"
	"viper/internal/hardware"
	"viper/internal/nvidia"
	"viper/internal/wmi"
)

// The names of the abilities the service reports.
const (
	capVersion                    = "VERSION"
	capOperatingMode              = "SUPPORT_OPERATING_MODE_CAPABILITY"
	capAudioMode                  = "SUPPORT_AUDIO_MODE"
	capGPUMode                    = "SUPPORT_GPU_MODE"
	capGPUModeCapability          = "SUPPORT_GPU_MODE_CAPABILITY"
	capLCDOverdrive               = "SUPPORT_LCD_OVERDRIVE"
	capCustomBootLogo             = "SUPPORT_CUSTOM_BOOT_LOGO"
	capKeyboardInfo               = "SUPPORT_KEYBOARD_INFO"
	capFanControl                 = "SUPPORT_FAN_CONTROL"
	capCPUInfo                    = "SUPPORT_CPU_INFO"
	capGPUInfo                    = "SUPPORT_GPU_INFO"
	capOverclockingModeCapability = "SUPPORT_OVERCLOCKING_MODE_CAPABILITY"
)

// machine is the set of abilities of the machine that the service uses.
//
// The service answers the official application, and it only needs the
// settings that the application asks for. Depending on the small
// interfaces instead of the whole of the machine keeps the service
// independent of the abilities it does not report, and it makes the
// stand for the machine in the tests that much smaller.
type machine interface {
	hardware.PowerModeReader
	hardware.LightingReader
	hardware.LightingWriter
	hardware.FansReader
	hardware.SwitchesReader
	// The service reports the readings of the machine, but it never asks
	// for the path of the gaming functions.
	Sensors() (hardware.Sensors, error)
	// The graphics switch is read and written under the name of the
	// display mode by the service, so it only needs those two calls of
	// the graphics mode.
	GPUMode() (hardware.GPUMode, error)
	SetGPUMode(mode hardware.GPUMode) error
	// Close releases the connection to the machine.
	Close()
}

// Provider collects the numbers that the service reports.
//
// The provider is filled with what the machine can tell. Everything that
// needs the gaming provider is skipped when the provider is not there, so
// the service also works without the rights of an administrator.
type Provider struct {
	backend machine
	store   *config.Store

	// mutex guards the settings that the service remembers itself.
	mutex sync.Mutex
	// remembered holds the settings that the firmware cannot give back.
	remembered rememberedSettings
}

// NewProvider builds a provider.
//
// A gaming provider that is a nil pointer inside the interface is taken for
// a missing one. An interface that holds a nil pointer is not equal to nil,
// so a guard that only asks the interface would let the call through and
// the call would crash the service.
func NewProvider(backend machine, store *config.Store) *Provider {
	return &Provider{backend: usableBackend(backend), store: store}
}

// usableBackend takes a gaming provider that cannot be used for nothing.
func usableBackend(backend machine) machine {
	if backend == nil {
		return nil
	}

	value := reflect.ValueOf(backend)
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		if value.IsNil() {
			return nil
		}
	}
	return backend
}

// hasBackend reports whether the gaming provider can be used.
//
// The interface alone does not tell: an interface that holds a nil pointer
// is not equal to nil, so the pointer itself has to be looked at. Without
// that look the first request that needs the machine would crash the
// service instead of reporting that the machine is out of reach.
func (p *Provider) hasBackend() bool {
	return usableBackend(p.backend) != nil
}

// Capability answers a request about the abilities of the machine.
//
// The second result is false when the service does not know the ability.
func (p *Provider) Capability(function string) (any, bool) {
	switch function {
	case capVersion:
		return ServiceVersion, true
	case capOperatingMode:
		return []any{0, 1, 4, 5, 6}, true
	case capAudioMode:
		return []any{
			"MUSIC", "MOVIE", "VOICE", "STRATEGY", "RPG", "SHOOTER", "CUSTOM_AUDIO", "AUTOMATIC",
		}, true
	case capGPUMode:
		return []any{0}, true
	case capGPUModeCapability:
		return []any{7}, true
	case capLCDOverdrive:
		return []any{1}, true
	case capCustomBootLogo:
		return []any{0}, true
	case capKeyboardInfo:
		return map[string]any{
			"keyboard_backlight_color_control": 2,
			"keyboard_language":                3,
			"keyboard_led_type":                4,
			"lightbar_type":                    2,
		}, true
	case capFanControl:
		return []any{
			fanAbility("CPU"),
			fanAbility("GPU"),
		}, true
	case capCPUInfo:
		return []any{
			map[string]any{
				"cpu_index":         0,
				"cpu_max_frequency": maximumClock(),
				"cpu_name":          processorName(),
			},
		}, true
	case capGPUInfo:
		return []any{
			map[string]any{
				"gpu_index":         0,
				"gpu_max_frequency": graphicsCardMaxClock(),
				"gpu_name":          graphicsCardName(),
			},
		}, true
	case capOverclockingModeCapability:
		// The original service does not implement this ability either. Its
		// answer carries no data at all, only the code of the failure, so
		// the server writes that answer on its own.
		return nil, false
	default:
		return nil, false
	}
}

// fanAbility describes one fan of the machine.
func fanAbility(name string) map[string]any {
	return map[string]any{
		"fan_custom_max_speed": 100,
		"fan_custom_min_speed": 0,
		"fan_index":            0,
		"fan_name":             name,
	}
}

// Setting answers a request about one setting.
//
// The second result is false when the service does not know the setting.
func (p *Provider) Setting(function string) (any, bool) {
	switch function {
	case "OPERATING_MODE":
		mode, err := p.operatingMode()
		if err != nil {
			return nil, false
		}
		return map[string]any{"mode": mode}, true
	case "GPU_MODE":
		mode, err := p.gpuMode()
		if err != nil {
			return nil, false
		}
		return map[string]any{"mode": mode}, true
	case "SOUND_MODE":
		// The sound profiles belong to the sound chip of the vendor, the
		// gaming provider has no interface for them. The service reports
		// its own record, see the comment of rememberedSettings.
		function, mode := p.soundModeState()
		return map[string]any{"function": function, "mode": mode}, true
	case "STICKY_KEY":
		enabled, err := hardware.StickyKeys()
		if err != nil {
			return nil, false
		}
		return map[string]any{"status": boolStatus(enabled)}, true
	case "BACKLIGHT_30S":
		enabled, err := p.backlightTimeout()
		if err != nil {
			return nil, false
		}
		return map[string]any{"status": boolStatus(enabled)}, true
	case "WIN_KEY":
		// The service tells whether the Windows key may be used, so the
		// number is one when the key is not locked.
		locked, err := p.winKeyLocked()
		if err != nil {
			return nil, false
		}
		return map[string]any{"status": boolStatus(!locked)}, true
	case "LCD_OVERDRIVE":
		enabled, err := p.lcdOverdrive()
		if err != nil {
			return nil, false
		}
		return map[string]any{"status": boolStatus(enabled)}, true
	case "BOOT_SOUND":
		enabled, err := p.bootSound()
		if err != nil {
			return nil, false
		}
		return map[string]any{"status": boolStatus(enabled)}, true
	case "ADAPTOR_STATUS":
		power, err := hardware.SystemPowerStatus()
		if err != nil {
			return nil, false
		}
		return map[string]any{"status": boolStatus(power.OnAC)}, true
	case "BATTERY_BOOST":
		// The battery boost lets the machine run faster on the battery. The
		// firmware has no interface for it, so the service reports its own
		// record, see the comment of rememberedSettings.
		return map[string]any{"status": p.batteryBoostState()}, true
	case "FAN_CONTROL":
		state, err := p.fans()
		if err != nil {
			return nil, false
		}
		return map[string]any{
			"mode": 0,
			"custom_fan_data": []any{
				fanData("CPU", state.CPUMode, state.CPUPercent),
				fanData("GPU", state.GPUMode, state.GPUPercent),
			},
		}, true
	case "LIGHTING":
		return p.lighting(), true
	case "CUSTOM_BOOT_LOGO":
		// The boot logo is a picture that the firmware of this machine does
		// not hold, so the service reports its own record, see the comment
		// of rememberedSettings.
		return map[string]any{"imageSource": p.bootLogoState()}, true
	default:
		return nil, false
	}
}

// fanData describes the state of one fan.
func fanData(name string, mode hardware.FanMode, percent int) map[string]any {
	auto := 1
	if mode != hardware.FanAuto {
		auto = 0
	}
	return map[string]any{
		"fan_custom_auto":  auto,
		"fan_custom_speed": percent,
		"fan_index":        0,
		"fan_name":         name,
	}
}

// boolStatus turns a switch into the number the service uses.
func boolStatus(enabled bool) int {
	if enabled {
		return 1
	}
	return 0
}

// operatingMode reads the performance profile of the machine.
func (p *Provider) operatingMode() (int, error) {
	if !p.hasBackend() {
		return 0, errNoBackend
	}
	mode, err := p.backend.PowerMode()
	if err != nil {
		return 0, err
	}
	switch mode {
	case hardware.PowerQuiet:
		return 0, nil
	case hardware.PowerBalanced:
		return 1, nil
	case hardware.PowerPerformance:
		return 4, nil
	case hardware.PowerTurbo:
		return 5, nil
	case hardware.PowerEco:
		return 6, nil
	default:
		return 1, nil
	}
}

// gpuMode reads the graphics switch of the machine.
//
// The service counts the modes from zero, the provider counts them from
// one.
func (p *Provider) gpuMode() (int, error) {
	if !p.hasBackend() {
		return 0, errNoBackend
	}
	mode, err := p.backend.GPUMode()
	if err != nil {
		return 0, err
	}
	if mode < hardware.GPUModeHybrid {
		return 0, nil
	}
	return int(mode) - 1, nil
}

// backlightTimeout tells whether the backlight switches off on its own.
func (p *Provider) backlightTimeout() (bool, error) {
	if !p.hasBackend() {
		return false, errNoBackend
	}
	state, err := p.backend.Backlight()
	if err != nil {
		return false, err
	}
	return state.Timeout, nil
}

// winKeyLocked tells whether the Windows key is locked.
func (p *Provider) winKeyLocked() (bool, error) {
	if !p.hasBackend() {
		return false, errNoBackend
	}
	return p.backend.WinKeyLocked()
}

// lcdOverdrive tells whether the display overdrive is active.
func (p *Provider) lcdOverdrive() (bool, error) {
	if !p.hasBackend() {
		return false, errNoBackend
	}
	return p.backend.LCDOverdrive()
}

// bootSound tells whether the sound at the start of the machine is on.
func (p *Provider) bootSound() (bool, error) {
	if !p.hasBackend() {
		return false, errNoBackend
	}
	return p.backend.BootSound()
}

// fans reads the state of the fans.
func (p *Provider) fans() (hardware.FanState, error) {
	if !p.hasBackend() {
		return hardware.FanState{}, errNoBackend
	}
	return p.backend.Fans()
}

// lighting describes the keyboard lighting.
//
// The firmware does not report the colours, so they come from the settings
// of Viper.
//
// 本机实测过：写完四个分区的颜色之后，GetGamingLED 读回来的字节仍然是
// 全零，GetGamingLEDColor 与 GetGamingLEDBehavior 恒答 2，八种灯效的
// 结果一模一样。让原厂服务自己设一次同样的颜色，它的应答里有四个分区，
// 但同一时刻用 WMI 读固件依然是全零——固件不通过 WMI 暴露分区颜色，
// 原厂软件也是自己记住的。Viper 与原厂行为等价。
func (p *Provider) lighting() map[string]any {
	result := map[string]any{
		"LEDs":       []any{},
		"brightness": 5,
		"color":      "#00aec7",
		"device":     0,
		"direction":  1,
		"duration":   3,
		"effect":     "WAVE",
		"speed":      3,
		"subindex":   map[string]any{"1": "WAVE"},
	}
	if p.store == nil {
		return result
	}

	stored := p.store.Settings().Lighting
	leds := make([]any, 0, len(stored.Zones))
	for i, colour := range stored.Zones {
		leds = append(leds, map[string]any{
			"LED_id": i,
			"color":  fmt.Sprintf("#%02x%02x%02x", colour.R, colour.G, colour.B),
			"status": 1,
		})
	}

	result["LEDs"] = leds
	result["brightness"] = stored.Brightness
	result["direction"] = stored.Direction
	result["duration"] = stored.Speed
	result["speed"] = stored.Speed
	result["effect"] = lightingEffectName(stored.Effect)
	result["subindex"] = map[string]any{"1": lightingEffectName(stored.Effect)}
	if len(stored.Zones) > 0 {
		colour := stored.Zones[0]
		result["color"] = fmt.Sprintf("#%02x%02x%02x", colour.R, colour.G, colour.B)
	}
	return result
}

// lightingEffectName gives the name of a lighting pattern.
func lightingEffectName(value uint64) string {
	switch hardware.LightingEffect(value) {
	case hardware.EffectStatic:
		return "STATIC"
	case hardware.EffectBreathing:
		return "BREATHING"
	case hardware.EffectNeon:
		return "NEON"
	case hardware.EffectWave:
		return "WAVE"
	case hardware.EffectSwitch:
		return "SHIFTING"
	case hardware.EffectScale:
		return "ZOOM"
	case hardware.EffectMeteor:
		return "METEOR"
	case hardware.EffectFlash:
		return "TWINKLING"
	default:
		return "STATIC"
	}
}

// Devices describes the devices that carry a lighting.
//
// The shape is the one of the profile of the original software. That
// service lists the keyboard twice, once for each of its two interfaces,
// so the list of Viper looks the same.
func (p *Provider) Devices() []any {
	device := func(index int) map[string]any {
		return map[string]any{
			"active_mode_index": 4,
			"brightness":        5,
			"device_index":      index,
			"device_name":       "AcerECKeyboard Device",
			"device_type":       5,
			"vector_of_modes":   lightingModes(),
		}
	}
	return []any{device(0), device(1)}
}

// lightingModes describes the lighting patterns of the keyboard.
//
// The numbers are the ones of the profile of the original software, they
// were read from its file.
func lightingModes() []any {
	type mode struct {
		name      string
		value     int
		flags     int
		colorMode int
		minColors int
		maxColors int
		minSpeed  int
		maxSpeed  int
		speed     int
		direction int
	}

	modes := []mode{
		{"DIRECT", 255, 48, 1, 0, 0, 0, 0, 0, 4294967295},
		{"STATIC", 0, 81, 2, 1, 1, 1, 9, 5, 0},
		{"BREATHING", 1, 81, 2, 1, 1, 1, 9, 5, 0},
		{"NEON", 2, 17, 0, 0, 0, 1, 9, 5, 0},
		{"WAVE", 3, 31, 0, 0, 0, 1, 9, 5, 0},
		{"SHIFTING", 4, 83, 2, 1, 1, 1, 9, 5, 0},
		{"ZOOM", 5, 81, 2, 1, 1, 1, 9, 5, 0},
		{"METEOR", 6, 81, 2, 1, 1, 1, 9, 5, 0},
		{"TWINKLING", 7, 81, 2, 1, 1, 1, 9, 5, 0},
	}

	out := make([]any, 0, len(modes))
	for _, item := range modes {
		out = append(out, map[string]any{
			"color_mode":               item.colorMode,
			"direction":                item.direction,
			"maximum_number_of_colors": item.maxColors,
			"maximum_speed":            item.maxSpeed,
			"minimum_speed":            item.minSpeed,
			"minimun_number_of_colors": item.minColors,
			"mode_flags":               item.flags,
			"mode_name":                item.name,
			"mode_value":               item.value,
			"speed_value":              item.speed,
		})
	}
	return out
}

// LightingEffects describes the lighting patterns in the short form.
func (p *Provider) LightingEffects() []any {
	return []any{
		map[string]any{"name": "DIRECT", "value": 255},
		map[string]any{"name": "STATIC", "value": 0},
		map[string]any{"name": "BREATHING", "value": 1},
		map[string]any{"name": "NEON", "value": 2},
		map[string]any{"name": "WAVE", "value": 3},
		map[string]any{"name": "SHIFTING", "value": 4},
		map[string]any{"name": "ZOOM", "value": 5},
		map[string]any{"name": "METEOR", "value": 6},
		map[string]any{"name": "TWINKLING", "value": 7},
	}
}

// MonitorData collects the live numbers of the machine.
func (p *Provider) MonitorData() map[string]any {
	data := map[string]any{
		"CPU_CORE_AMOUNT":    processorCount(),
		"CPU_CORE_FREQUENCY": []any{},
		"CPU_FREQUENCY":      float64(processorClock()),
		"CPU_MAX_FREQUENCY":  maximumClock(),
		"CPU_USAGE":          cpuUsage(),
		"ETHERNET_DOWNLOAD":  0.0,
		"ETHERNET_UPLOAD":    0.0,
		"WIFI_DOWNLOAD":      0.0,
		"WIFI_UPLOAD":        0.0,
		"RAM_FREQUENCY":      memoryFrequency(),
	}

	total, used := memoryInfo()
	data["RAM_TOTAL"] = float64(total) / (1 << 20)
	data["RAM_USAGE"] = used

	if p.hasBackend() {
		if sensors, err := p.backend.Sensors(); err == nil {
			data["CPU_TEMPERATURE"] = sensors.CPUTempC
			data["CPU_FANSPEED"] = sensors.CPUFanRPM
			data["SYS1_TEMPERATURE"] = sensors.SystemTempC
			data["SYS1_FANSPEED"] = sensors.CPUFanRPM
			data["GPU1_TEMPERATURE"] = sensors.GPUTempC
			data["GPU1_FANSPEED"] = sensors.GPUFanRPM
		}
	}

	if cards, err := nvidia.Read(); err == nil && len(cards) > 0 {
		card := cards[0]
		data["GPU1_USAGE"] = card.UsagePercent
		data["GPU1_FREQUENCY"] = card.GraphicsClockMHz
		data["GPU1_MAX_FREQUENCY"] = card.MaxGraphicsClockMHz
		// The original service tells whether the card sits in a low
		// performance state, a card that runs at a small part of its
		// highest clock is idle.
		idle := 0
		if card.MaxGraphicsClockMHz > 0 && card.GraphicsClockMHz*5 < card.MaxGraphicsClockMHz {
			idle = 1
		}
		data["GPU1_IDLE_STATUS"] = idle
	}

	return data
}

// maximumClock reads the highest clock of the processor.
//
// The number comes from the standard namespace of Windows. That namespace
// only knows the nominal clock of the processor, the original service
// reports the clock of the turbo boost, which Windows does not offer
// through an interface. The clock the machine runs at is never higher than
// the number that is reported.
//
// The function is a variable, so that a test can put a number in place of
// the call to the namespace, which is slow and needs the rights of an
// administrator on some machines.
var maximumClock = func() int {
	clock := processorClock()

	client, err := wmi.Connect(wmi.NamespaceCIM)
	if err != nil {
		return clock
	}
	defer client.Close()

	paths, err := client.Instances("Win32_Processor")
	if err != nil || len(paths) == 0 {
		return clock
	}
	value, err := client.ReadProperty(paths[0], "MaxClockSpeed")
	if err != nil {
		return clock
	}
	if int(value) > clock {
		return int(value)
	}
	return clock
}

// memoryFrequency reads the clock of the memory.
func memoryFrequency() float64 {
	speed, err := hardware.MemoryFrequency()
	if err != nil {
		return 0
	}
	return float64(speed)
}

// graphicsCardName reads the name of the graphics card.
func graphicsCardName() string {
	if cards, err := nvidia.Read(); err == nil && len(cards) > 0 && cards[0].Name != "" {
		return cards[0].Name
	}
	return ""
}

// graphicsCardMaxClock reads the highest clock of the graphics card.
//
// The original service fills this field with the clock of the processor,
// which is wrong: the field belongs to the graphics card. The driver of
// the card knows its own highest clock, so that number is used. A machine
// without a card of the vendor reports zero.
//
// 原厂服务把这个字段填成处理器的频率，字段的名字却是显卡的。显卡驱动知道
// 自己能跑多高，所以用显卡自己的数字；没有显卡的机器报零。
func graphicsCardMaxClock() int {
	cards, err := nvidia.Read()
	if err != nil || len(cards) == 0 {
		return 0
	}
	return cards[0].MaxGraphicsClockMHz
}
