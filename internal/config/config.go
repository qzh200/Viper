// Package config keeps the settings of Viper between runs.
//
// The settings live in a single JSON file below the user profile, so that
// the state of the keyboard lighting and of the fans can be reported even
// though the firmware does not report the lighting state back.
//
// 文件里存的是硬件层枚举的语义值，也就是 hardware.PowerMode 和
// hardware.FanMode 的编号，永远不是固件要的编码。固件的性能档位编码是
// 0、1、4、5、6，风扇行为编码是 1、2、3，它们只存在于硬件层里面。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// 机器的性能档位。
//
// 这里的值是硬件层枚举（hardware.PowerMode）的编号：静音、均衡、性能、
// 加速、未知、节能。它们不是固件的编码，固件的编码是 0、1、4、5、6。
const (
	// PowerQuiet favours silence over performance.
	PowerQuiet uint64 = iota
	// PowerBalanced is the profile that Viper starts with.
	PowerBalanced
	// PowerPerformance favours performance.
	PowerPerformance
	// PowerTurbo is the maximum profile, available on Predator models only.
	PowerTurbo
	// PowerUnknown is a profile the firmware does not describe. It keeps
	// the number of the hardware layer, so that the two enumerations can
	// be compared.
	PowerUnknown
	// PowerEco is the profile that saves power.
	PowerEco
)

// 一个风扇组的行为。
//
// 这里的值是硬件层枚举（hardware.FanMode）的编号：自动、最大转速、固定
// 占空比。它们不是固件的编码，固件的编码是 1、2、3。
const (
	// FanAuto lets the embedded controller follow its own curve.
	FanAuto uint64 = iota
	// FanMax drives the fan at full speed.
	FanMax
	// FanCustom drives the fan at a fixed duty cycle.
	FanCustom
)

// Lighting is the remembered keyboard lighting.
type Lighting struct {
	// Effect is the lighting pattern.
	Effect uint64 `json:"effect"`
	// Speed is the animation speed.
	Speed uint64 `json:"speed"`
	// Direction is the animation direction.
	Direction uint64 `json:"direction"`
	// Brightness is the overall brightness.
	Brightness uint64 `json:"brightness"`
	// Zones holds the colour of each zone.
	Zones [4]LightingColor `json:"zones"`
	// PerZone reports whether the zones carry individual colours.
	PerZone bool `json:"perZone"`
}

// LightingColor is the colour of one lighting zone.
type LightingColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// Fans is the remembered fan configuration.
type Fans struct {
	// CPUMode is the behaviour of the CPU fan. It is the semantic
	// enumeration of the hardware layer (hardware.FanMode), not the
	// encoding that the firmware expects.
	//
	// 语义枚举，不是固件编码。
	CPUMode uint64 `json:"cpuMode"`
	// GPUMode is the behaviour of the GPU fan. It is the semantic
	// enumeration of the hardware layer (hardware.FanMode), not the
	// encoding that the firmware expects.
	//
	// 语义枚举，不是固件编码。
	GPUMode uint64 `json:"gpuMode"`
	// CPUPercent is the duty cycle of the CPU fan.
	CPUPercent int `json:"cpuPercent"`
	// GPUPercent is the duty cycle of the GPU fan.
	GPUPercent int `json:"gpuPercent"`
	// CoolBoost reports whether the boost table is selected.
	CoolBoost bool `json:"coolBoost"`
}

// Scenario is a named set of settings.
//
// The scenarios carry the lighting and the fans, so that every
// performance profile can have its own look. The original software stores
// them in the same way, it calls them scenes.
type Scenario struct {
	// Lighting is the keyboard lighting of the scenario.
	Lighting Lighting `json:"lighting"`
	// Fans is the fan configuration of the scenario.
	Fans Fans `json:"fans"`
}

// Settings is the complete state of Viper.
type Settings struct {
	// Lighting is the keyboard lighting.
	Lighting Lighting `json:"lighting"`
	// Fans is the fan configuration.
	Fans Fans `json:"fans"`
	// PowerMode is the performance profile. It is the semantic
	// enumeration of the hardware layer (hardware.PowerMode), not the
	// encoding that the firmware expects.
	//
	// 语义枚举，不是固件编码。
	PowerMode uint64 `json:"powerMode"`
	// BacklightTimeout reports whether the backlight switches off.
	BacklightTimeout bool `json:"backlightTimeout"`
	// Scenarios holds the saved scenes, indexed by their name.
	Scenarios map[string]Scenario `json:"scenarios"`
	// Bindings ties a performance profile to a scene, the key is the
	// name of the profile and the value the name of the scene.
	Bindings map[string]string `json:"bindings"`
}

// copy returns a copy of the settings that shares nothing with the
// original.
//
// Only the maps of the scenes and of the bindings carry a reference, the
// rest of the settings is a value.
//
// 副本和原设置不共享任何东西，调用方改动副本不会改到存储里的设置。
func (s Settings) copy() Settings {
	out := s
	out.Scenarios = make(map[string]Scenario, len(s.Scenarios))
	for name, scenario := range s.Scenarios {
		out.Scenarios[name] = scenario
	}
	out.Bindings = make(map[string]string, len(s.Bindings))
	for profile, scene := range s.Bindings {
		out.Bindings[profile] = scene
	}
	return out
}

// Store reads and writes the settings.
//
// The service of Viper shares one store between the connections of its
// clients, so the store is guarded against concurrent use: a change and a
// write never run at the same time and a reader never sees a half
// written file.
//
// Viper 的服务在多个连接之间共用一个 Store，所以这个 Store 是加锁的：
// 改动和写盘不会同时发生，读到的文件也不会是写了一半的。
type Store struct {
	// mu guards the settings and the file.
	mu sync.RWMutex
	// path is the location of the settings file.
	path string
	// settings are the settings that are kept.
	settings Settings
}

// Default returns the settings Viper starts with.
func Default() Settings {
	return Settings{
		Lighting: Lighting{
			Effect:     0,
			Speed:      3,
			Direction:  0,
			Brightness: 50,
			PerZone:    false,
		},
		Fans: Fans{
			CPUMode:    FanAuto,
			GPUMode:    FanAuto,
			CPUPercent: 0,
			GPUPercent: 0,
		},
		PowerMode:        PowerBalanced,
		BacklightTimeout: true,
		Scenarios:        map[string]Scenario{},
		Bindings:         map[string]string{},
	}
}

// Open loads the settings from the user profile.
//
// A missing file is not an error, the defaults are used instead.
func Open() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return OpenAt(filepath.Join(dir, "Viper", "config.json"))
}

// OpenAt loads the settings from the given file.
//
// A missing file is not an error, the defaults are used instead.
func OpenAt(path string) (*Store, error) {
	store := &Store{path: path, settings: Default()}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return store, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &store.settings); err != nil {
		// A broken file must not stop the program, the defaults are used.
		store.settings = Default()
	}
	// An older file has no scenes, the maps have to exist anyway.
	if store.settings.Scenarios == nil {
		store.settings.Scenarios = map[string]Scenario{}
	}
	if store.settings.Bindings == nil {
		store.settings.Bindings = map[string]string{}
	}
	return store, nil
}

// Settings returns the current settings.
//
// The settings are a copy: the maps that they carry are not shared with
// the store, so a caller cannot change the stored settings by accident.
//
// 返回的是深拷贝，调用方拿到的是自己的副本。
func (s *Store) Settings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.copy()
}

// Path returns the location of the settings file.
func (s *Store) Path() string { return s.path }

// Save writes the settings back.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

// save writes the settings back, the caller holds the lock.
//
// The file is written beside the settings and moved over them, so that a
// reader never sees a file that is only half written.
//
// 文件先写在旁边再移过去，读的人不会读到写了一半的文件。
func (s *Store) save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encode: %w", err)
	}
	data = append(data, '\n')

	temp, err := os.CreateTemp(dir, "config-*.json")
	if err != nil {
		return fmt.Errorf("config: 在 %s 里建临时文件失败: %w", dir, err)
	}
	name := temp.Name()
	// The temporary file is only removed when the move over the settings
	// did not happen.
	defer os.Remove(name)

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("config: 写 %s 失败: %w", name, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("config: 写 %s 失败: %w", name, err)
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return fmt.Errorf("config: 设置 %s 的权限失败: %w", name, err)
	}
	if err := os.Rename(name, s.path); err != nil {
		return fmt.Errorf("config: 写 %s 失败: %w", s.path, err)
	}
	return nil
}

// Update applies a change to the settings and stores them.
//
// The change runs while the store is locked, so that two changes cannot
// overwrite each other. The change must not read the store again.
//
// 改动在锁里进行，两个改动不会互相覆盖。改动函数里不要再读这个 Store。
func (s *Store) Update(change func(*Settings)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	change(&s.settings)
	return s.save()
}
