package model

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"viper/internal/domain"
)

type Level int

const (
	LevelNormal Level = iota
	LevelWarn
	LevelDanger
)

type Metric struct {
	Label   string
	Value   string
	Number  float64
	Detail  string
	Percent float64
	Level   Level
}

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

func (o Overview) Metric(label string) Metric {
	for _, metric := range o.Metrics {
		if metric.Label == label {
			return metric
		}
	}
	return Metric{}
}

type Overview struct {
	Connected        bool
	Route            string
	Instance         string
	Error            string
	PowerMode        domain.PowerMode
	PowerModes       []domain.PowerMode
	FanState         domain.FanState
	Lighting         domain.RGBConfig
	BacklightTimeout bool
	GPUMode          domain.GPUMode
	DiscreteGPU      bool
	LCDOverdrive     bool
	WinKeyLocked     bool
	BootSound        bool
	Metrics          []Metric
	Monitor          domain.MonitorData
	History          []Sample
	Updated          time.Time
}

const HistoryLimit = 10

type Sample struct {
	Time       time.Time
	CPUTempC   int
	GPUTempC   int
	BoardTempC int
	CPUUsage   float64
	GPUUsage   float64
}

const (
	WarmTemperature = 70
	HotTemperature  = 85
	WarmLoad        = 70
	HotLoad         = 90
)

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

func FormatCelsius(celsius int) string {
	return strconv.Itoa(celsius) + " °C"
}

func FormatPercent(percent float64) string {
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}
	return strconv.FormatFloat(percent, 'f', 1, 64) + "%"
}

func FormatMHz(mhz int) string {
	if mhz <= 0 {
		return "—"
	}
	if mhz < 1000 {
		return strconv.Itoa(mhz) + " MHz"
	}
	return strconv.FormatFloat(float64(mhz)/1000, 'f', 1, 64) + " GHz"
}

func FormatMemoryClock(mhz int) string {
	if mhz <= 0 {
		return "—"
	}
	return strconv.Itoa(mhz) + " MHz"
}

func FormatMB(mb float64) string {
	if mb <= 0 {
		return "—"
	}
	return strconv.FormatFloat(mb/1024, 'f', 1, 64) + " GB"
}

func FormatRPM(rpm int) string {
	return strconv.Itoa(rpm) + " RPM"
}

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

func PowerModeLabels(modes []domain.PowerMode) []string {
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		out = append(out, FormatPowerMode(mode))
	}
	return out
}

func PowerModeIndex(modes []domain.PowerMode, current domain.PowerMode) int {
	for i, mode := range modes {
		if mode == current {
			return i
		}
	}
	return 0
}

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

func FanModeLabels() []string {
	return []string{"自动", "全速", "自定义"}
}

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

type LightingEffectItem struct {
	Effect domain.LightingEffect
	Name   string
	Desc   string
}

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

func LightingEffectLabels() []string {
	effects := LightingEffects()
	labels := make([]string, len(effects))
	for i, e := range effects {
		labels[i] = e.Name
	}
	return labels
}

func LightingEffectIndex(effect domain.LightingEffect) int {
	for i, item := range LightingEffects() {
		if item.Effect == effect {
			return i
		}
	}
	return 0
}

func LightingEffectFromIndex(index int) domain.LightingEffect {
	effects := LightingEffects()
	if index >= 0 && index < len(effects) {
		return effects[index].Effect
	}
	return domain.EffectStatic
}

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

type PresetColor struct {
	Name  string
	Color domain.RGBColor
}

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

func HueToRGB(hue float64) domain.RGBColor {
	return HSVToRGB(hue, 1.0, 1.0)
}

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

func RGBToHue(col domain.RGBColor) float64 {
	h, _, _ := RGBToHSV(col)
	return h
}

func LightingEffectCanColor(effect domain.LightingEffect) bool {
	return effect != domain.EffectNeon && effect != domain.EffectWave
}

func LightingEffectCanSpeed(effect domain.LightingEffect) bool {
	return effect != domain.EffectStatic
}

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

func GPUModeLabels() []string {
	return []string{"混合输出", "独显直连", "自动切换"}
}

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
