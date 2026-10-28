package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/hardware"
)

// Scenarios 返回所有已保存的命名场景配置字典。
func (a *App) Scenarios() map[string]config.Scenario {
	return a.store.Settings().Scenarios
}

// Bindings 返回所有电源模式与场景的绑定关联映射。
func (a *App) Bindings() map[string]string {
	return a.store.Settings().Bindings
}

// SaveScenario 将当前的灯效与风扇状态保存为一个新的命名场景。
func (a *App) SaveScenario(name string) error {
	if name == "" {
		return fmt.Errorf("app: 场景需要一个名字")
	}

	settings := a.store.Settings()
	scenario := config.Scenario{
		Lighting: settings.Lighting,
		Fans:     settings.Fans,
	}

	if lighting, err := a.Lighting(); err == nil {
		stored := toStoredLighting(lighting)
		if stored.Brightness == 0 {
			stored.Brightness = settings.Lighting.Brightness
		}
		scenario.Lighting = stored
	}

	if state, err := a.Fans(); err == nil {
		scenario.Fans = config.Fans{
			CPUMode:    storedFanMode(state.CPUMode),
			GPUMode:    storedFanMode(state.GPUMode),
			CPUPercent: state.CPUPercent,
			GPUPercent: state.GPUPercent,
			CoolBoost:  state.CoolBoost,
		}
	}

	if err := a.store.Update(func(s *config.Settings) {
		s.Scenarios[name] = scenario
	}); err != nil {
		return fmt.Errorf("app: 保存场景 %q 失败: %w", name, err)
	}
	return nil
}

// DeleteScenario 删除指定名称的场景，并自动解除相关电源模式绑定。
func (a *App) DeleteScenario(name string) error {
	if err := a.store.Update(func(s *config.Settings) {
		delete(s.Scenarios, name)
		for profile, scene := range s.Bindings {
			if scene == name {
				delete(s.Bindings, profile)
			}
		}
	}); err != nil {
		return fmt.Errorf("app: 删除场景 %q 失败: %w", name, err)
	}
	return nil
}

// ApplyScenario 根据名称加载并应用指定的场景配置。
func (a *App) ApplyScenario(name string) error {
	scenario, ok := a.store.Settings().Scenarios[name]
	if !ok {
		return fmt.Errorf("app: 没有叫 %q 的场景", name)
	}
	return a.applyScenario(scenario)
}

func (a *App) applyScenario(scenario config.Scenario) error {
	lighting := hardware.RGBConfig{
		Effect:     hardware.LightingEffect(scenario.Lighting.Effect),
		Speed:      scenario.Lighting.Speed,
		Direction:  scenario.Lighting.Direction,
		Brightness: scenario.Lighting.Brightness,
		PerZone:    scenario.Lighting.PerZone,
	}
	for i, colour := range scenario.Lighting.Zones {
		lighting.Zones[i] = hardware.RGBColor{R: colour.R, G: colour.G, B: colour.B}
	}
	if err := a.SetLighting(lighting); err != nil {
		return err
	}

	return a.SetFans(hardware.FanConfig{
		CPUMode:    fanMode(scenario.Fans.CPUMode),
		GPUMode:    fanMode(scenario.Fans.GPUMode),
		CPUPercent: scenario.Fans.CPUPercent,
		GPUPercent: scenario.Fans.GPUPercent,
		CoolBoost:  scenario.Fans.CoolBoost,
	})
}

// BindScenario 将指定的场景绑定到特定电源性能模式，在切换到该模式时自动应用该场景。
func (a *App) BindScenario(profile, scene string) error {
	if _, ok := a.store.Settings().Scenarios[scene]; !ok {
		return fmt.Errorf("app: 没有叫 %q 的场景", scene)
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.Bindings[profile] = scene
	}); err != nil {
		return fmt.Errorf("app: 保存档位 %q 的场景绑定失败: %w", profile, err)
	}
	return nil
}

// UnbindScenario 解除特定电源性能模式的场景绑定。
func (a *App) UnbindScenario(profile string) error {
	if err := a.store.Update(func(s *config.Settings) {
		delete(s.Bindings, profile)
	}); err != nil {
		return fmt.Errorf("app: 解绑档位 %q 的场景失败: %w", profile, err)
	}
	return nil
}
