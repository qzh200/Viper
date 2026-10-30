// Package model 定义界面展示用的数据模型。
//
// 这里的每个结构体都是给界面看的：它们由硬件读数转换而来，
// 不含任何图形代码，因此可以单独、完整地测试。
package model

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"viper/internal/domain"
)

// Level 是指标的严重程度，决定它的配色。
type Level int

const (
	// LevelNormal 表示指标在正常范围内。
	LevelNormal Level = iota
	// LevelWarn 表示指标偏高，值得留意。
	LevelWarn
	// LevelDanger 表示指标到了危险范围。
	LevelDanger
)

// Metric 是首页上的一个指标块。
type Metric struct {
	// Label 是指标的名字，如「CPU 温度」。
	Label string
	// Value 是指标的读数，如「72 °C」。
	Value string
	// Number 是读数的数值部分，画表与风扇时用。
	Number float64
	// Detail 是指标的补充说明，如「占用 43.2% · 3.4 GHz」。
	Detail string
	// Percent 是指标归一化后的位置，画进度条用，取值 0 到 1；
	// 没有上下限可比的指标留 0。
	Percent float64
	// Level 是指标的严重程度。
	Level Level
}

// Label 返回严重程度在界面上显示的名字。
func (l Level) Label() string {
	switch l {
	case LevelWarn:
		return "偏高"
	case LevelDanger:
		return "过高"
	default:
		return "正常"
	}
}

// Metric 按名字找出一个指标；没有就返回空的。
func (o Overview) Metric(label string) Metric {
	for _, metric := range o.Metrics {
		if metric.Label == label {
			return metric
		}
	}
	return Metric{}
}

// Overview 是首页一次刷新的全部内容。
type Overview struct {
	// Connected 表示是否连上了原厂服务。
	Connected bool
	// Route 是当前连接通道的名字。
	Route string
	// Instance 是后端驱动实例的名字。
	Instance string
	// Error 是最近一次失败的原因，成功时为空。
	Error string
	// PowerMode 是当前生效的性能档位。
	PowerMode domain.PowerMode
	// PowerModes 是可选的性能档位。
	PowerModes []domain.PowerMode
	// FanState 是当前风扇工作模式与强冷配置。
	FanState domain.FanState
	// Lighting 是当前键盘 RGB 灯效配置。
	Lighting domain.RGBConfig
	// BacklightTimeout 表示背光是否在 30 秒空闲时自动熄灭。
	BacklightTimeout bool
	// GPUMode 是当前生效的显卡工作模式。
	GPUMode domain.GPUMode
	// DiscreteGPU 标记当前是否处于纯独显直连状态。
	DiscreteGPU bool
	// LCDOverdrive 标记屏幕响应时间超速加速是否开启。
	LCDOverdrive bool
	// WinKeyLocked 标记 Windows 徽标键是否处于禁用锁定状态。
	WinKeyLocked bool
	// Metrics 是首页展示的指标。
	Metrics []Metric
	// Monitor 是这次读到的整机遥测，界面要用那些没变成指标的数字。
	Monitor domain.MonitorData
	// History 是最近的采样，界面上用来画曲线。
	History []Sample
	// Updated 是这次读数的时间。
	Updated time.Time
}

// HistoryLimit 是界面保留的采样个数：曲线画最近这么多点。
//
// 采样两秒一次，十个点就是最近二十秒。
const HistoryLimit = 10

// Sample 是一次采样，界面上用来画曲线。
type Sample struct {
	// Time 是采样的时刻。
	Time time.Time
	// CPUTempC 是 CPU 温度。
	CPUTempC int
	// GPUTempC 是 GPU 温度。
	GPUTempC int
	// BoardTempC 是主板温度。
	BoardTempC int
	// CPUUsage 是 CPU 负载百分比。
	CPUUsage float64
	// GPUUsage 是 GPU 负载百分比。
	GPUUsage float64
}

// 温度与负载的分档线，与 Predator / Nitro 的散热设计对齐。
const (
	// WarmTemperature 是温度开始需要留意的度数。
	WarmTemperature = 70
	// HotTemperature 是温度到达危险范围的度数。
	HotTemperature = 85
	// WarmLoad 是负载开始需要留意的百分比。
	WarmLoad = 70
	// HotLoad 是负载到达危险范围的百分比。
	HotLoad = 90
)

// TemperatureLevel 按温度给出严重程度。
func TemperatureLevel(celsius int) Level {
	switch {
	case celsius >= HotTemperature:
		return LevelDanger
	case celsius >= WarmTemperature:
		return LevelWarn
	default:
		return LevelNormal
	}
}

// LoadLevel 按负载百分比给出严重程度。
func LoadLevel(percent float64) Level {
	switch {
	case percent >= HotLoad:
		return LevelDanger
	case percent >= WarmLoad:
		return LevelWarn
	default:
		return LevelNormal
	}
}

// Metrics 把整机遥测转换成一组指标，顺序固定，界面照着画就行。
func Metrics(monitor domain.MonitorData) []Metric {
	return []Metric{
		{
			Label:   "CPU 温度",
			Value:   FormatCelsius(monitor.CPUTemperatureC),
			Number:  float64(monitor.CPUTemperatureC),
			Detail:  strings.Join([]string{FormatPercent(monitor.CPUUsagePercent), FormatMHz(monitor.CPUFrequencyMHz)}, " · "),
			Percent: clamp01(monitor.CPUUsagePercent / 100),
			Level:   TemperatureLevel(monitor.CPUTemperatureC),
		},
		{
			Label:   "GPU 温度",
			Value:   FormatCelsius(monitor.GPUTemperatureC),
			Number:  float64(monitor.GPUTemperatureC),
			Detail:  strings.Join([]string{FormatPercent(float64(monitor.GPUUsagePercent)), FormatMHz(monitor.GPUFrequencyMHz)}, " · "),
			Percent: clamp01(float64(monitor.GPUUsagePercent) / 100),
			Level:   TemperatureLevel(monitor.GPUTemperatureC),
		},
		{
			Label:   "内存占用",
			Value:   FormatPercent(monitor.RAMUsagePercent),
			Number:  monitor.RAMUsagePercent,
			Detail:  FormatMB(monitor.RAMTotalMB) + " @ " + FormatMemoryClock(monitor.RAMFrequencyMHz),
			Percent: clamp01(monitor.RAMUsagePercent / 100),
			Level:   LoadLevel(monitor.RAMUsagePercent),
		},
		{
			Label:   "CPU 风扇",
			Value:   FormatRPM(monitor.CPUFanRPM),
			Number:  float64(monitor.CPUFanRPM),
			Detail:  FormatMHz(monitor.CPUFrequencyMHz),
			Percent: 0,
			Level:   LevelNormal,
		},
		{
			Label:   "GPU 风扇",
			Value:   FormatRPM(monitor.GPUFanRPM),
			Number:  float64(monitor.GPUFanRPM),
			Detail:  FormatMHz(monitor.GPUFrequencyMHz),
			Percent: 0,
			Level:   LevelNormal,
		},
		{
			Label:   "主板温度",
			Value:   FormatCelsius(monitor.SystemTempC),
			Number:  float64(monitor.SystemTempC),
			Detail:  "供电与主板",
			Percent: 0,
			Level:   TemperatureLevel(monitor.SystemTempC),
		},
	}
}

// FormatCelsius 把温度写成「72 °C」。
func FormatCelsius(celsius int) string {
	return strconv.Itoa(celsius) + " °C"
}

// FormatPercent 把百分比写成「43.2%」，自动钳制在 0% 到 100% 之间。
func FormatPercent(percent float64) string {
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}
	return strconv.FormatFloat(percent, 'f', 1, 64) + "%"
}

// FormatMHz 把频率写成「3.4 GHz」，不足 1000 MHz 时写「800 MHz」。
func FormatMHz(mhz int) string {
	if mhz <= 0 {
		return "—"
	}
	if mhz < 1000 {
		return strconv.Itoa(mhz) + " MHz"
	}
	return strconv.FormatFloat(float64(mhz)/1000, 'f', 1, 64) + " GHz"
}

// FormatMemoryClock 把内存频率写成「3200 MHz」：内存的频率习惯用 MHz 讲，
// 不像核心频率那样换算成 GHz。
func FormatMemoryClock(mhz int) string {
	if mhz <= 0 {
		return "—"
	}
	return strconv.Itoa(mhz) + " MHz"
}

// FormatMB 把内存容量写成「15.9 GB」。
func FormatMB(mb float64) string {
	if mb <= 0 {
		return "—"
	}
	return strconv.FormatFloat(mb/1024, 'f', 1, 64) + " GB"
}

// FormatRPM 把转速写成「4200 RPM」。
func FormatRPM(rpm int) string {
	return strconv.Itoa(rpm) + " RPM"
}

// FormatKbps 把网络速率写成「1.2 Mbps」。
func FormatKbps(kbps float64) string {
	switch {
	case kbps <= 0:
		return "0 Kbps"
	case kbps < 1000:
		return strconv.FormatFloat(kbps, 'f', 0, 64) + " Kbps"
	default:
		return strconv.FormatFloat(kbps/1000, 'f', 1, 64) + " Mbps"
	}
}

// FormatPowerMode 把性能档位写成界面上的名字。
func FormatPowerMode(mode domain.PowerMode) string {
	switch mode {
	case domain.PowerQuiet:
		return "静音"
	case domain.PowerBalanced:
		return "平衡"
	case domain.PowerPerformance:
		return "性能"
	case domain.PowerTurbo:
		return "狂暴"
	case domain.PowerEco:
		return "节能"
	default:
		return "未知"
	}
}

// PowerModeLabels 把可选的性能档位写成界面上的名字，顺序与档位一致。
func PowerModeLabels(modes []domain.PowerMode) []string {
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		out = append(out, FormatPowerMode(mode))
	}
	return out
}

// PowerModeIndex 找出当前档位在可选档位里的位置；不在其中时返回 0。
func PowerModeIndex(modes []domain.PowerMode, current domain.PowerMode) int {
	for i, mode := range modes {
		if mode == current {
			return i
		}
	}
	return 0
}

// StatusLine 写出连接状态那一行：连上了说通道，没连上说原因。
func StatusLine(overview Overview) string {
	if overview.Connected {
		route := overview.Route
		if route == "" {
			route = "原厂服务"
		}
		return fmt.Sprintf("已连接 · %s", route)
	}
	if overview.Error != "" {
		return "未连接 · " + overview.Error
	}
	return "未连接"
}

// FanModeLabels 返回风扇模式的中文选项标签。
func FanModeLabels() []string {
	return []string{"自动", "全速", "自定义"}
}

// FanModeIndex 返回给定模式在 FanModeLabels 里的下标。
func FanModeIndex(mode domain.FanMode) int {
	switch mode {
	case domain.FanMax:
		return 1
	case domain.FanCustom:
		return 2
	default:
		return 0
	}
}

// FanModeFromIndex 将下标还原为 FanMode。
func FanModeFromIndex(index int) domain.FanMode {
	switch index {
	case 1:
		return domain.FanMax
	case 2:
		return domain.FanCustom
	default:
		return domain.FanAuto
	}
}

// clamp01 把比例限制在 0 到 1 之间。
func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

// LightingEffectItem 描述一个灯效模式的名称和说明。
type LightingEffectItem struct {
	Effect domain.LightingEffect
	Name   string
	Desc   string
}

// LightingEffects 返回支持的键盘动态灯效列表。
//
// 顺序与宏碁官方灯效面板一致：静态、呼吸、霓虹、波形、切换、缩放、流星、闪烁。
func LightingEffects() []LightingEffectItem {
	return []LightingEffectItem{
		{Effect: domain.EffectStatic, Name: "静态", Desc: "经典全域单色常亮，专注不刺眼"},
		{Effect: domain.EffectBreathing, Name: "呼吸", Desc: "明暗起伏如呼吸节奏，静谧电竞氛围"},
		{Effect: domain.EffectNeon, Name: "霓虹", Desc: "全光谱平滑循环变色，流光溢彩"},
		{Effect: domain.EffectWave, Name: "波形", Desc: "彩虹光波横向穿梭流动，动感十足"},
		{Effect: domain.EffectSwitch, Name: "切换", Desc: "双色交替渐变，呼吸感十足"},
		{Effect: domain.EffectScale, Name: "缩放", Desc: "中心向两端缩放扩张与收缩光波"},
		{Effect: domain.EffectMeteor, Name: "流星雨", Desc: "高速流星拖尾自一侧划过键盘"},
		{Effect: domain.EffectFlash, Name: "闪烁", Desc: "节奏跃动爆闪，激情电竞战意"},
	}
}

// LightingEffectLabels 返回所有灯效的简短名称。
func LightingEffectLabels() []string {
	effects := LightingEffects()
	labels := make([]string, len(effects))
	for i, e := range effects {
		labels[i] = e.Name
	}
	return labels
}

// LightingEffectIndex 返回指定灯效在 LightingEffects 列表中的下标。
func LightingEffectIndex(effect domain.LightingEffect) int {
	for i, item := range LightingEffects() {
		if item.Effect == effect {
			return i
		}
	}
	return 0
}

// LightingEffectFromIndex 根据下标返回对应的灯效模式。
func LightingEffectFromIndex(index int) domain.LightingEffect {
	effects := LightingEffects()
	if index >= 0 && index < len(effects) {
		return effects[index].Effect
	}
	return domain.EffectStatic
}

// ZoneName 返回 4 分区键盘中每个分区的中文名字。
func ZoneName(zoneIndex int) string {
	switch zoneIndex {
	case 0:
		return "分区 1 (WASD/左侧)"
	case 1:
		return "分区 2 (主键区中左)"
	case 2:
		return "分区 3 (主键区右侧/方向)"
	case 3:
		return "分区 4 (数字小键盘区)"
	default:
		return fmt.Sprintf("分区 %d", zoneIndex+1)
	}
}

// PresetColor 预设颜色定义。
type PresetColor struct {
	Name  string
	Color domain.RGBColor
}

// LightingPresetColors 返回快捷精选色盘（12款经典高饱和电竞色彩）。
func LightingPresetColors() []PresetColor {
	return []PresetColor{
		{Name: "掠夺蓝", Color: domain.RGBColor{R: 0, G: 174, B: 243}},
		{Name: "极光青", Color: domain.RGBColor{R: 0, G: 230, B: 118}},
		{Name: "烈焰红", Color: domain.RGBColor{R: 255, G: 23, B: 68}},
		{Name: "赛博紫", Color: domain.RGBColor{R: 213, G: 0, B: 249}},
		{Name: "炽热橙", Color: domain.RGBColor{R: 255, G: 109, B: 0}},
		{Name: "日光金", Color: domain.RGBColor{R: 255, G: 234, B: 0}},
		{Name: "深邃蓝", Color: domain.RGBColor{R: 0, G: 112, B: 243}},
		{Name: "珊瑚粉", Color: domain.RGBColor{R: 255, G: 64, B: 129}},
		{Name: "薄荷绿", Color: domain.RGBColor{R: 29, G: 233, B: 182}},
		{Name: "冰纯白", Color: domain.RGBColor{R: 255, G: 255, B: 255}},
		{Name: "幽灵灰", Color: domain.RGBColor{R: 90, G: 105, B: 120}},
		{Name: "熄灭黑", Color: domain.RGBColor{R: 0, G: 0, B: 0}},
	}
}

// HueToRGB 根据色相角度（0~360度）计算对应的全饱和高亮 RGB。
func HueToRGB(hue float64) domain.RGBColor {
	return HSVToRGB(hue, 1.0, 1.0)
}

// HSVToRGB 将 Hue(0~360), Saturation(0~1), Value(0~1) 转换为 RGB 颜色。
func HSVToRGB(h, s, v float64) domain.RGBColor {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	if s < 0 {
		s = 0
	} else if s > 1 {
		s = 1
	}
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}

	c := v * s
	x := c * (1.0 - math.Abs(math.Mod(h/60.0, 2.0)-1.0))
	m := v - c

	var r1, g1, b1 float64
	switch {
	case h < 60:
		r1, g1, b1 = c, x, 0
	case h < 120:
		r1, g1, b1 = x, c, 0
	case h < 180:
		r1, g1, b1 = 0, c, x
	case h < 240:
		r1, g1, b1 = 0, x, c
	case h < 300:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}

	return domain.RGBColor{
		R: uint8(math.Round((r1 + m) * 255)),
		G: uint8(math.Round((g1 + m) * 255)),
		B: uint8(math.Round((b1 + m) * 255)),
	}
}

// RGBToHSV 将 RGB 转换为精确的 Hue(0~360), Saturation(0~1), Value(0~1)。
func RGBToHSV(col domain.RGBColor) (h, s, v float64) {
	r := float64(col.R) / 255.0
	g := float64(col.G) / 255.0
	b := float64(col.B) / 255.0

	maxVal := math.Max(r, math.Max(g, b))
	minVal := math.Min(r, math.Min(g, b))
	delta := maxVal - minVal

	v = maxVal
	if maxVal > 1e-4 {
		s = delta / maxVal
	} else {
		s = 0
	}

	if delta < 1e-4 {
		h = 0
	} else {
		switch maxVal {
		case r:
			h = math.Mod((g-b)/delta, 6)
		case g:
			h = (b-r)/delta + 2
		case b:
			h = (r-g)/delta + 4
		}
		h *= 60
		if h < 0 {
			h += 360
		}
	}
	return h, s, v
}

// RGBToHue 计算 RGB 颜色的主色相角度（0~360度）。
func RGBToHue(col domain.RGBColor) float64 {
	h, _, _ := RGBToHSV(col)
	return h
}

// LightingEffectCanColor 判断灯效模式是否允许调整颜色。
// 霓虹与波形由全光谱流动驱动，不支持自定义颜色；其余模式均可调色。
func LightingEffectCanColor(effect domain.LightingEffect) bool {
	return effect != domain.EffectNeon && effect != domain.EffectWave
}

// LightingEffectCanSpeed 判断灯效模式是否允许调节动画速度。
// 静态模式为常亮无动画，其余动态模式均可调速。
func LightingEffectCanSpeed(effect domain.LightingEffect) bool {
	return effect != domain.EffectStatic
}

// PowerModeDesc 返回性能档位的简要特性说明。
func PowerModeDesc(mode domain.PowerMode) string {
	switch mode {
	case domain.PowerQuiet:
		return "风扇静音运行，严格控制发热，适合图书馆与静音办公"
	case domain.PowerBalanced:
		return "日常使用的综合平衡方案，功耗与噪音智能自适应"
	case domain.PowerPerformance:
		return "解锁 CPU 与 GPU 功耗释放墙，激进散热保障高帧率"
	case domain.PowerTurbo:
		return "满血功耗与满载超频状态，双风扇拉满压制极限热量"
	case domain.PowerEco:
		return "最大化电池续航与节能，限制峰值核心频率与功耗"
	default:
		return "系统默认性能调校策略"
	}
}

// FormatGPUMode 返回显卡工作模式的中文名称。
func FormatGPUMode(mode domain.GPUMode) string {
	switch mode {
	case domain.GPUModeDiscrete:
		return "独显直连"
	case domain.GPUModeAuto:
		return "自动切换"
	default:
		return "混合输出"
	}
}

// GPUModeLabels 返回显卡模式的选项标签。
func GPUModeLabels() []string {
	return []string{"混合输出", "独显直连", "自动切换"}
}

// GPUModeIndex 返回给定显卡模式在 GPUModeLabels 里的下标。
func GPUModeIndex(mode domain.GPUMode) int {
	switch mode {
	case domain.GPUModeDiscrete:
		return 1
	case domain.GPUModeAuto:
		return 2
	default:
		return 0
	}
}

// GPUModeFromIndex 将下标还原为 GPUMode。
func GPUModeFromIndex(index int) domain.GPUMode {
	switch index {
	case 1:
		return domain.GPUModeDiscrete
	case 2:
		return domain.GPUModeAuto
	default:
		return domain.GPUModeHybrid
	}
}

// GPUModeDesc 返回显卡工作模式的说明。
func GPUModeDesc(mode domain.GPUMode) string {
	switch mode {
	case domain.GPUModeDiscrete:
		return "独显直接连接内屏与外接屏幕，绕过核显减少延迟，电竞游戏帧率更高"
	case domain.GPUModeAuto:
		return "由系统与驱动根据运行应用与负载动态智能调度显卡通道"
	default:
		return "独显渲染并由核显输出屏幕，日常使用与续航更持久，兼顾能效"
	}
}
