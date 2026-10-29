// Package model 定义界面展示用的数据模型。
//
// 这里的每个结构体都是给界面看的：它们由硬件读数转换而来，
// 不含任何图形代码，因此可以单独、完整地测试。
package model

import (
	"fmt"
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

// FormatPercent 把百分比写成「43.2%」。
func FormatPercent(percent float64) string {
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
