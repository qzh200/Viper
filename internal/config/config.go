// Package config 提供用户配置、灯效预设、风扇策略与场景绑定的持久化存储。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// 电源模式数值常量（对应 domain.PowerMode 编码）。
const (
	PowerQuiet uint64 = iota
	PowerBalanced
	PowerPerformance
	PowerTurbo
	PowerUnknown
	PowerEco
)

// 风扇模式数值常量。
const (
	FanAuto uint64 = iota
	FanMax
	FanCustom
)

// Lighting 记录持久化的键盘灯效设置。
type Lighting struct {
	Effect     uint64           `json:"effect"`
	Speed      uint64           `json:"speed"`
	Direction  uint64           `json:"direction"`
	Brightness uint64           `json:"brightness"`
	Zones      [4]LightingColor `json:"zones"`
	PerZone    bool             `json:"perZone"`
}

// LightingColor 记录分区 RGB 颜色。
type LightingColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// Fans 记录持久化的风扇配置。
type Fans struct {
	CPUMode    uint64 `json:"cpuMode"`
	GPUMode    uint64 `json:"gpuMode"`
	CPUPercent int    `json:"cpuPercent"`
	GPUPercent int    `json:"gpuPercent"`
	CoolBoost  bool   `json:"coolBoost"`
}

// Scenario 组合特定场景下的灯效与风扇设定。
type Scenario struct {
	Lighting Lighting `json:"lighting"`
	Fans     Fans     `json:"fans"`
}

// Settings 承载应用程序的全部持久化配置信息。
type Settings struct {
	Lighting         Lighting            `json:"lighting"`
	Fans             Fans                `json:"fans"`
	PowerMode        uint64              `json:"powerMode"`
	BacklightTimeout bool                `json:"backlightTimeout"`
	Scenarios        map[string]Scenario `json:"scenarios"`
	Bindings         map[string]string   `json:"bindings"`
}

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

// Store 线程安全地管理配置文件的读取、修改与原子写入。
type Store struct {
	mu       sync.RWMutex
	path     string
	settings Settings
}

// Default 返回系统的默认出厂设置。
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

// Open 打开默认用户配置目录下的配置文件（%APPDATA%/Viper/config.json）。
func Open() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return OpenAt(filepath.Join(dir, "Viper", "config.json"))
}

// OpenAt 打开指定文件路径下的配置文件，如果不存在则使用默认配置。
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
		store.settings = Default()
	}

	if store.settings.Scenarios == nil {
		store.settings.Scenarios = map[string]Scenario{}
	}
	if store.settings.Bindings == nil {
		store.settings.Bindings = map[string]string{}
	}
	return store, nil
}

// Settings 返回当前配置的快照拷贝。
func (s *Store) Settings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.copy()
}

// Path 返回当前配置文件的物理存储路径。
func (s *Store) Path() string { return s.path }

// Save 将内存中的配置原子刷写到磁盘文件。
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

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

// Update 在排他锁保护下执行闭包修改并持久化到文件。
func (s *Store) Update(change func(*Settings)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	change(&s.settings)
	return s.save()
}
