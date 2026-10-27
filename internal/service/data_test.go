package service

import (
	"path/filepath"
	"testing"

	"viper/internal/config"
	"viper/internal/hardware"
)

// The answers of the service have to match the ones of the original
// software, so the tests below pin down the names and the shapes of the
// abilities that the official application asks for.

func TestCapabilityOfTheVersion(t *testing.T) {
	value, ok := NewProvider(nil, nil).Capability(capVersion)
	if !ok {
		t.Fatal("the version is not known")
	}
	if value != ServiceVersion {
		t.Errorf("the version is %v, want %s", value, ServiceVersion)
	}
}

func TestCapabilityOfTheOperatingModes(t *testing.T) {
	value, ok := NewProvider(nil, nil).Capability(capOperatingMode)
	if !ok {
		t.Fatal("the operating modes are not known")
	}
	modes, ok := value.([]any)
	if !ok {
		t.Fatalf("the operating modes are %T, want a list", value)
	}
	want := []any{0, 1, 4, 5, 6}
	if len(modes) != len(want) {
		t.Fatalf("the operating modes are %v, want %v", modes, want)
	}
	for i := range want {
		if modes[i] != want[i] {
			t.Errorf("the operating mode %d is %v, want %v", i, modes[i], want[i])
		}
	}
}

func TestCapabilityOfTheAudioModes(t *testing.T) {
	value, ok := NewProvider(nil, nil).Capability(capAudioMode)
	if !ok {
		t.Fatal("the audio modes are not known")
	}
	modes, ok := value.([]any)
	if !ok {
		t.Fatalf("the audio modes are %T, want a list", value)
	}
	if len(modes) != 8 {
		t.Fatalf("the audio modes are %v, want eight of them", modes)
	}
	if modes[0] != "MUSIC" || modes[7] != "AUTOMATIC" {
		t.Errorf("the audio modes are %v, want the list of the original software", modes)
	}
}

func TestCapabilityOfTheKeyboard(t *testing.T) {
	value, ok := NewProvider(nil, nil).Capability(capKeyboardInfo)
	if !ok {
		t.Fatal("the keyboard is not known")
	}
	info, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("the keyboard is %T, want a set of values", value)
	}
	cases := map[string]int{
		"keyboard_backlight_color_control": 2,
		"keyboard_language":                3,
		"keyboard_led_type":                4,
		"lightbar_type":                    2,
	}
	for name, want := range cases {
		if info[name] != want {
			t.Errorf("the keyboard reports %v for %s, want %d", info[name], name, want)
		}
	}
}

func TestCapabilityOfTheFans(t *testing.T) {
	value, ok := NewProvider(nil, nil).Capability(capFanControl)
	if !ok {
		t.Fatal("the fans are not known")
	}
	fans, ok := value.([]any)
	if !ok {
		t.Fatalf("the fans are %T, want a list", value)
	}
	if len(fans) != 2 {
		t.Fatalf("the fans are %v, want two of them", fans)
	}
	for i, name := range []string{"CPU", "GPU"} {
		fan, ok := fans[i].(map[string]any)
		if !ok {
			t.Fatalf("the fan %d is %T, want a set of values", i, fans[i])
		}
		if fan["fan_name"] != name {
			t.Errorf("the fan %d is called %v, want %s", i, fan["fan_name"], name)
		}
		if fan["fan_custom_max_speed"] != 100 || fan["fan_custom_min_speed"] != 0 {
			t.Errorf("the fan %d reports the speeds %v and %v, want 100 and 0",
				i, fan["fan_custom_max_speed"], fan["fan_custom_min_speed"])
		}
	}
}

func TestCapabilityOfAnUnknownAbility(t *testing.T) {
	value, ok := NewProvider(nil, nil).Capability("SUPPORT_NOTHING")
	if ok {
		t.Error("an unknown ability was accepted")
	}
	if value != nil {
		t.Errorf("an unknown ability carries %v, want nothing", value)
	}
}

func TestFanAbility(t *testing.T) {
	fan := fanAbility("CPU")
	if fan["fan_index"] != 0 {
		t.Errorf("the fan has the index %v, want 0", fan["fan_index"])
	}
	if fan["fan_name"] != "CPU" {
		t.Errorf("the fan is called %v, want CPU", fan["fan_name"])
	}
}

func TestFanDataOfTheAutomaticMode(t *testing.T) {
	// In the automatic mode the embedded controller runs its own curve, so
	// the service has to tell the client that the value is not a fixed
	// duty cycle.
	fan := fanData("CPU", hardware.FanAuto, 45)
	if fan["fan_custom_auto"] != 1 {
		t.Errorf("the fan reports %v for the automatic switch, want 1", fan["fan_custom_auto"])
	}
	if fan["fan_custom_speed"] != 45 {
		t.Errorf("the fan reports the speed %v, want 45", fan["fan_custom_speed"])
	}
	if fan["fan_name"] != "CPU" {
		t.Errorf("the fan is called %v, want CPU", fan["fan_name"])
	}
}

func TestFanDataOfTheFixedModes(t *testing.T) {
	for _, mode := range []hardware.FanMode{hardware.FanMax, hardware.FanCustom} {
		fan := fanData("GPU", mode, 60)
		if fan["fan_custom_auto"] != 0 {
			t.Errorf("the fan reports %v for the automatic switch, want 0", fan["fan_custom_auto"])
		}
		if fan["fan_custom_speed"] != 60 {
			t.Errorf("the fan reports the speed %v, want 60", fan["fan_custom_speed"])
		}
	}
}

func TestBoolStatus(t *testing.T) {
	if boolStatus(true) != 1 {
		t.Errorf("the switched on state is %d, want 1", boolStatus(true))
	}
	if boolStatus(false) != 0 {
		t.Errorf("the switched off state is %d, want 0", boolStatus(false))
	}
}

func TestLightingEffectName(t *testing.T) {
	// The names are the ones that the profile of the original software
	// uses, they are not the short names of the hardware layer.
	cases := map[hardware.LightingEffect]string{
		hardware.EffectStatic:    "STATIC",
		hardware.EffectBreathing: "BREATHING",
		hardware.EffectNeon:      "NEON",
		hardware.EffectWave:      "WAVE",
		hardware.EffectSwitch:    "SHIFTING",
		hardware.EffectScale:     "ZOOM",
		hardware.EffectMeteor:    "METEOR",
		hardware.EffectFlash:     "TWINKLING",
	}
	for effect, want := range cases {
		if got := lightingEffectName(uint64(effect)); got != want {
			t.Errorf("the effect %d renders as %q, want %q", effect, got, want)
		}
	}
}

func TestLightingEffectNameOfAnUnknownEffect(t *testing.T) {
	// The original service falls back to the static pattern for a value
	// that it does not know.
	if got := lightingEffectName(99); got != "STATIC" {
		t.Errorf("an unknown effect renders as %q, want STATIC", got)
	}
}

func TestLightingModes(t *testing.T) {
	modes := lightingModes()
	if len(modes) != 9 {
		t.Fatalf("the lighting modes are %d, want nine of them", len(modes))
	}
	first, ok := modes[0].(map[string]any)
	if !ok {
		t.Fatalf("the first lighting mode is %T, want a set of values", modes[0])
	}
	if first["mode_name"] != "DIRECT" {
		t.Errorf("the first lighting mode is called %v, want DIRECT", first["mode_name"])
	}
	if first["mode_value"] != 255 {
		t.Errorf("the first lighting mode has the value %v, want 255", first["mode_value"])
	}
	last, ok := modes[len(modes)-1].(map[string]any)
	if !ok {
		t.Fatalf("the last lighting mode is %T, want a set of values", modes[len(modes)-1])
	}
	if last["mode_name"] != "TWINKLING" {
		t.Errorf("the last lighting mode is called %v, want TWINKLING", last["mode_name"])
	}
	if last["mode_value"] != 7 {
		t.Errorf("the last lighting mode has the value %v, want 7", last["mode_value"])
	}
}

func TestLightingEffects(t *testing.T) {
	effects := NewProvider(nil, nil).LightingEffects()
	if len(effects) != 9 {
		t.Fatalf("the lighting effects are %d, want nine of them", len(effects))
	}
	first, ok := effects[0].(map[string]any)
	if !ok {
		t.Fatalf("the first lighting effect is %T, want a set of values", effects[0])
	}
	if first["name"] != "DIRECT" || first["value"] != 255 {
		t.Errorf("the first lighting effect is %v, want DIRECT with the value 255", first)
	}
}

func TestDevices(t *testing.T) {
	// The original service lists the keyboard twice, once for each of its
	// two interfaces.
	devices := NewProvider(nil, nil).Devices()
	if len(devices) != 2 {
		t.Fatalf("the devices are %d, want two of them", len(devices))
	}
	for i, entry := range devices {
		device, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("the device %d is %T, want a set of values", i, entry)
		}
		if device["device_index"] != i {
			t.Errorf("the device %d has the index %v, want %d", i, device["device_index"], i)
		}
		if device["device_name"] != "AcerECKeyboard Device" {
			t.Errorf("the device %d is called %v, want AcerECKeyboard Device", i, device["device_name"])
		}
	}
}

func TestLightingWithoutAStore(t *testing.T) {
	// Without the settings of Viper the service reports the values of the
	// original software, they are the defaults of the keyboard.
	lighting := NewProvider(nil, nil).lighting()
	if lighting["effect"] != "WAVE" {
		t.Errorf("the effect is %v, want WAVE", lighting["effect"])
	}
	if lighting["brightness"] != 5 {
		t.Errorf("the brightness is %v, want 5", lighting["brightness"])
	}
	if lighting["color"] != "#00aec7" {
		t.Errorf("the colour is %v, want #00aec7", lighting["color"])
	}
	leds, ok := lighting["LEDs"].([]any)
	if !ok {
		t.Fatalf("the zones are %T, want a list", lighting["LEDs"])
	}
	if len(leds) != 0 {
		t.Errorf("the zones are %v, want none of them", leds)
	}
}

func TestLightingWithAStore(t *testing.T) {
	// The firmware does not report the colours, so the service has to
	// answer with the ones that Viper remembers.
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *config.Settings) {
		s.Lighting.Effect = uint64(hardware.EffectWave)
		s.Lighting.Speed = 5
		s.Lighting.Brightness = 70
		s.Lighting.Zones[0] = config.LightingColor{R: 199, G: 0, B: 135}
		s.Lighting.Zones[1] = config.LightingColor{R: 0, G: 174, B: 199}
	}); err != nil {
		t.Fatal(err)
	}

	lighting := NewProvider(nil, store).lighting()
	if lighting["effect"] != "WAVE" {
		t.Errorf("the effect is %v, want WAVE", lighting["effect"])
	}
	if lighting["brightness"] != uint64(70) {
		t.Errorf("the brightness is %v, want 70", lighting["brightness"])
	}
	if lighting["speed"] != uint64(5) || lighting["duration"] != uint64(5) {
		t.Errorf("the speed is %v and the duration is %v, want 5 and 5",
			lighting["speed"], lighting["duration"])
	}
	if lighting["color"] != "#c70087" {
		t.Errorf("the colour is %v, want the colour of the first zone", lighting["color"])
	}

	leds, ok := lighting["LEDs"].([]any)
	if !ok {
		t.Fatalf("the zones are %T, want a list", lighting["LEDs"])
	}
	if len(leds) != 4 {
		t.Fatalf("the zones are %d, want the four of them", len(leds))
	}
	first, ok := leds[0].(map[string]any)
	if !ok {
		t.Fatalf("the first zone is %T, want a set of values", leds[0])
	}
	if first["color"] != "#c70087" {
		t.Errorf("the first zone has the colour %v, want #c70087", first["color"])
	}
	if first["LED_id"] != 0 || first["status"] != 1 {
		t.Errorf("the first zone is %v, want the identifier 0 and the status 1", first)
	}
}

func TestSettingOfTheSound(t *testing.T) {
	// The sound profiles belong to the sound chip of the vendor, Viper
	// does not touch them.
	value, ok := NewProvider(nil, nil).Setting("SOUND_MODE")
	if !ok {
		t.Fatal("the sound mode is not known")
	}
	sound, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("the sound mode is %T, want a set of values", value)
	}
	if sound["function"] != "DTS" || sound["mode"] != 0 {
		t.Errorf("the sound mode is %v, want DTS with the mode 0", sound)
	}
}

func TestSettingOfTheBatteryBoost(t *testing.T) {
	value, ok := NewProvider(nil, nil).Setting("BATTERY_BOOST")
	if !ok {
		t.Fatal("the battery boost is not known")
	}
	boost, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("the battery boost is %T, want a set of values", value)
	}
	if boost["status"] != 1 {
		t.Errorf("the battery boost reports %v, want 1", boost["status"])
	}
}

func TestSettingOfTheBootLogo(t *testing.T) {
	value, ok := NewProvider(nil, nil).Setting("CUSTOM_BOOT_LOGO")
	if !ok {
		t.Fatal("the boot logo is not known")
	}
	logo, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("the boot logo is %T, want a set of values", value)
	}
	if logo["imageSource"] != "" {
		t.Errorf("the boot logo carries %v, want an empty text", logo["imageSource"])
	}
}

func TestSettingOfTheLighting(t *testing.T) {
	value, ok := NewProvider(nil, nil).Setting("LIGHTING")
	if !ok {
		t.Fatal("the lighting is not known")
	}
	lighting, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("the lighting is %T, want a set of values", value)
	}
	if lighting["effect"] != "WAVE" {
		t.Errorf("the effect is %v, want WAVE", lighting["effect"])
	}
	subindex, ok := lighting["subindex"].(map[string]any)
	if !ok {
		t.Fatalf("the subindex is %T, want a set of values", lighting["subindex"])
	}
	if subindex["1"] != "WAVE" {
		t.Errorf("the subindex holds %v, want WAVE", subindex["1"])
	}
}

func TestSettingWithoutTheGamingProvider(t *testing.T) {
	// Without the rights of an administrator the settings that need the
	// gaming provider cannot be answered. The service has to say so
	// instead of pretending that the setting is off.
	provider := NewProvider(nil, nil)
	names := []string{
		"OPERATING_MODE",
		"GPU_MODE",
		"WIN_KEY",
		"LCD_OVERDRIVE",
		"BOOT_SOUND",
		"FAN_CONTROL",
		"BACKLIGHT_30S",
	}
	for _, name := range names {
		if _, ok := provider.Setting(name); ok {
			t.Errorf("the setting %s was answered without the gaming provider", name)
		}
	}
}

func TestSettingOfAnUnknownName(t *testing.T) {
	value, ok := NewProvider(nil, nil).Setting("NOTHING")
	if ok {
		t.Error("an unknown setting was answered")
	}
	if value != nil {
		t.Errorf("an unknown setting carries %v, want nothing", value)
	}
}
