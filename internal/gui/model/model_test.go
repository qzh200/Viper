package model

import (
	"math"
	"testing"
	"time"

	"viper/internal/domain"
)

func TestMetricsCoverTheWholeMachine(t *testing.T) {
	monitor := domain.MonitorData{
		CPUTemperatureC:    72,
		CPUUsagePercent:    43.2,
		CPUFrequencyMHz:    3400,
		CPUMaxFrequencyMHz: 4900,
		CPUFanRPM:          4200,
		GPUTemperatureC:    61,
		GPUUsagePercent:    12,
		GPUFrequencyMHz:    1980,
		GPUFanRPM:          2100,
		SystemTempC:        48,
		RAMTotalMB:         16384,
		RAMUsagePercent:    55.5,
		RAMFrequencyMHz:    3200,
	}

	metrics := Metrics(monitor)
	if len(metrics) != 6 {
		t.Fatalf("there are %d metrics, want 6", len(metrics))
	}

	labels := map[string]Metric{}
	for _, metric := range metrics {
		labels[metric.Label] = metric
	}

	cpu, ok := labels["CPU 温度"]
	if !ok {
		t.Fatalf("the CPU temperature is missing from %v", metrics)
	}
	if cpu.Value != "72 °C" {
		t.Errorf("the CPU temperature is %q, want \"72 °C\"", cpu.Value)
	}
	if cpu.Level != LevelWarn {
		t.Errorf("72 degrees is %v, want a warning", cpu.Level)
	}
	if math.Abs(cpu.Percent-0.432) > 1e-9 {
		t.Errorf("the CPU load is %v, want 0.432", cpu.Percent)
	}

	memory, ok := labels["内存占用"]
	if !ok {
		t.Fatalf("the memory is missing from %v", metrics)
	}
	if memory.Value != "55.5%" {
		t.Errorf("the memory use is %q, want \"55.5%%\"", memory.Value)
	}
	if memory.Detail != "16.0 GB @ 3200 MHz" {
		t.Errorf("the memory detail is %q, want \"16.0 GB @ 3200 MHz\"", memory.Detail)
	}
}

func TestMetricsClampTheLoadIntoABar(t *testing.T) {
	metrics := Metrics(domain.MonitorData{CPUUsagePercent: 140, RAMUsagePercent: -20})
	for _, metric := range metrics {
		if metric.Percent < 0 || metric.Percent > 1 {
			t.Errorf("the %s bar is at %v, want between 0 and 1", metric.Label, metric.Percent)
		}
	}
}

func TestTemperatureLevel(t *testing.T) {
	cases := map[int]Level{
		30:  LevelNormal,
		69:  LevelNormal,
		70:  LevelWarn,
		84:  LevelWarn,
		85:  LevelDanger,
		100: LevelDanger,
	}
	for celsius, want := range cases {
		if got := TemperatureLevel(celsius); got != want {
			t.Errorf("%d degrees is %v, want %v", celsius, got, want)
		}
	}
}

func TestLoadLevel(t *testing.T) {
	cases := map[float64]Level{
		0:    LevelNormal,
		69.9: LevelNormal,
		70:   LevelWarn,
		89.9: LevelWarn,
		90:   LevelDanger,
		100:  LevelDanger,
	}
	for percent, want := range cases {
		if got := LoadLevel(percent); got != want {
			t.Errorf("%v percent is %v, want %v", percent, got, want)
		}
	}
}

func TestFormatCelsius(t *testing.T) {
	if got := FormatCelsius(72); got != "72 °C" {
		t.Errorf("FormatCelsius(72) = %q, want \"72 °C\"", got)
	}
}

func TestFormatPercent(t *testing.T) {
	if got := FormatPercent(43.25); got != "43.2%" {
		t.Errorf("FormatPercent(43.25) = %q, want \"43.2%%\"", got)
	}
}

func TestFormatMHz(t *testing.T) {
	cases := map[int]string{
		0:    "—",
		800:  "800 MHz",
		1000: "1.0 GHz",
		3400: "3.4 GHz",
	}
	for mhz, want := range cases {
		if got := FormatMHz(mhz); got != want {
			t.Errorf("FormatMHz(%d) = %q, want %q", mhz, got, want)
		}
	}
}

func TestFormatMemoryClock(t *testing.T) {
	if got := FormatMemoryClock(3200); got != "3200 MHz" {
		t.Errorf("FormatMemoryClock(3200) = %q, want \"3200 MHz\"", got)
	}
	if got := FormatMemoryClock(0); got != "—" {
		t.Errorf("FormatMemoryClock(0) = %q, want a dash", got)
	}
}

func TestFormatMB(t *testing.T) {
	if got := FormatMB(16384); got != "16.0 GB" {
		t.Errorf("FormatMB(16384) = %q, want \"16.0 GB\"", got)
	}
	if got := FormatMB(0); got != "—" {
		t.Errorf("FormatMB(0) = %q, want a dash", got)
	}
}

func TestFormatRPM(t *testing.T) {
	if got := FormatRPM(4200); got != "4200 RPM" {
		t.Errorf("FormatRPM(4200) = %q, want \"4200 RPM\"", got)
	}
}

func TestFormatKbps(t *testing.T) {
	cases := map[float64]string{
		0:    "0 Kbps",
		12:   "12 Kbps",
		999:  "999 Kbps",
		1500: "1.5 Mbps",
	}
	for kbps, want := range cases {
		if got := FormatKbps(kbps); got != want {
			t.Errorf("FormatKbps(%v) = %q, want %q", kbps, got, want)
		}
	}
}

func TestTheLevelHasAName(t *testing.T) {
	cases := map[Level]string{
		LevelNormal: "正常",
		LevelWarn:   "偏高",
		LevelDanger: "过高",
	}
	for level, want := range cases {
		if got := level.Label(); got != want {
			t.Errorf("the name of %v is %q, want %q", level, got, want)
		}
	}
}

func TestAMetricIsFoundByItsName(t *testing.T) {
	overview := Overview{Metrics: Metrics(domain.MonitorData{CPUTemperatureC: 66})}

	cpu := overview.Metric("CPU 温度")
	if cpu.Value != "66 °C" {
		t.Errorf("the CPU temperature is %q, want \"66 °C\"", cpu.Value)
	}
	if cpu.Number != 66 {
		t.Errorf("the CPU temperature is %v, want 66", cpu.Number)
	}
	if missing := overview.Metric("没有这个"); missing.Label != "" {
		t.Errorf("a metric that is not there came back as %+v", missing)
	}
}

func TestFormatPowerMode(t *testing.T) {
	cases := map[domain.PowerMode]string{
		domain.PowerQuiet:       "静音",
		domain.PowerBalanced:    "平衡",
		domain.PowerPerformance: "性能",
		domain.PowerTurbo:       "狂暴",
		domain.PowerEco:         "节能",
		domain.PowerUnknown:     "未知",
	}
	for mode, want := range cases {
		if got := FormatPowerMode(mode); got != want {
			t.Errorf("the name of %q is %q, want %q", mode, got, want)
		}
	}
}

func TestPowerModeLabels(t *testing.T) {
	modes := []domain.PowerMode{domain.PowerQuiet, domain.PowerTurbo}
	labels := PowerModeLabels(modes)
	if len(labels) != 2 || labels[0] != "静音" || labels[1] != "狂暴" {
		t.Errorf("the labels are %q, want the quiet and the turbo mode", labels)
	}
}

func TestPowerModeIndex(t *testing.T) {
	modes := []domain.PowerMode{domain.PowerQuiet, domain.PowerBalanced, domain.PowerTurbo}
	if got := PowerModeIndex(modes, domain.PowerBalanced); got != 1 {
		t.Errorf("the balanced mode is at %d, want 1", got)
	}
	if got := PowerModeIndex(modes, domain.PowerEco); got != 0 {
		t.Errorf("a mode that is not there is at %d, want 0", got)
	}
}

func TestStatusLine(t *testing.T) {
	connected := Overview{Connected: true, Route: "原厂服务 AcerAgentService"}
	if got := StatusLine(connected); got != "已连接 · 原厂服务 AcerAgentService" {
		t.Errorf("the status is %q, want the connected line", got)
	}

	noRoute := Overview{Connected: true}
	if got := StatusLine(noRoute); got != "已连接 · 原厂服务" {
		t.Errorf("the status is %q, want the connected line with the default route", got)
	}

	broken := Overview{Error: "connection refused"}
	if got := StatusLine(broken); got != "未连接 · connection refused" {
		t.Errorf("the status is %q, want the failed line", got)
	}

	quiet := Overview{}
	if got := StatusLine(quiet); got != "未连接" {
		t.Errorf("the status is %q, want the plain failed line", got)
	}
}

func TestOverviewKeepsItsTime(t *testing.T) {
	overview := Overview{Updated: time.Now()}
	if overview.Updated.IsZero() {
		t.Error("the overview lost its time")
	}
}
