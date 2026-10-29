package model

import (
	"math"
	"strings"
	"sync"

	"viper/internal/domain"
)

// DefaultSmoothAlpha 是默认的平滑阻尼因子。
//
// 取 0.06 时，在 25ms 步进下，大约在 1000ms 左右平稳过渡到目标值，
// 带来沉稳高级的指针质感，彻底消除读数暴跌骤升的过山车感。
const DefaultSmoothAlpha = 0.06

// Smoother 为硬件监控读数提供低通滤波与缓动阻尼平滑，
// 杜绝读数大起大落剧烈跳跃，并自动清洗过滤宏碁原厂服务在休眠时返回的负数异常值（如 -1%）。
type Smoother struct {
	mu          sync.Mutex
	initialized bool

	cpuTemp   float64
	gpuTemp   float64
	boardTemp float64

	cpuUsage float64
	gpuUsage float64
	ramUsage float64

	cpuSamples []float64
	gpuSamples []float64

	cpuFan float64
	gpuFan float64

	cpuFreq float64
	gpuFreq float64

	wifiDown float64
	wifiUp   float64
	ethDown  float64
	ethUp    float64

	target domain.MonitorData
}

// NewSmoother 创建一个平滑缓冲器。
func NewSmoother() *Smoother {
	return &Smoother{}
}

// Initialized 返回平滑器是否已经加载过至少一次真实硬件读数。
func (s *Smoother) Initialized() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initialized
}

// SetTarget 设置最新的物理硬件数据。
//
// 第一次设置时会立即到位（不产生开机爬升动画）；
// 随后的变动会先经过异常值清洗与滑动窗口加权滤波，彻底消除单点瞬时突发的直角跳变。
func (s *Smoother) SetTarget(raw domain.MonitorData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clean := cleanMonitor(raw)
	if !s.initialized {
		s.snapLocked(clean)
		return
	}

	// 维护 CPU / GPU 负载的滑动采样窗口（最近 4 次硬件采样），做加权移动平均，
	// 彻底熨平 20%+ 瞬间砸到 0% 的离散跳跃，呈现温和的动态呼吸感：
	s.cpuSamples = append(s.cpuSamples, clean.CPUUsagePercent)
	if len(s.cpuSamples) > 4 {
		s.cpuSamples = s.cpuSamples[len(s.cpuSamples)-4:]
	}
	clean.CPUUsagePercent = weightedAverage(s.cpuSamples)

	s.gpuSamples = append(s.gpuSamples, float64(clean.GPUUsagePercent))
	if len(s.gpuSamples) > 4 {
		s.gpuSamples = s.gpuSamples[len(s.gpuSamples)-4:]
	}
	clean.GPUUsagePercent = int(math.Round(weightedAverage(s.gpuSamples)))

	// 网络速率：流量上升时积极跟随，流量下降时平缓衰减，杜绝数字在几百Kbps与0之间频繁闪烁：
	clean.WIFIDownloadKbps = smoothNetworkRate(s.target.WIFIDownloadKbps, clean.WIFIDownloadKbps)
	clean.WIFIUploadKbps = smoothNetworkRate(s.target.WIFIUploadKbps, clean.WIFIUploadKbps)
	clean.EthernetDownloadKbps = smoothNetworkRate(s.target.EthernetDownloadKbps, clean.EthernetDownloadKbps)
	clean.EthernetUploadKbps = smoothNetworkRate(s.target.EthernetUploadKbps, clean.EthernetUploadKbps)

	s.target = clean
}

// Snap 立即将所有指标对齐到目标硬件数据，不产生平滑过渡。
func (s *Smoother) Snap(raw domain.MonitorData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapLocked(cleanMonitor(raw))
}

func (s *Smoother) snapLocked(target domain.MonitorData) {
	s.initialized = true
	s.target = target
	s.cpuTemp = float64(target.CPUTemperatureC)
	s.gpuTemp = float64(target.GPUTemperatureC)
	s.boardTemp = float64(target.SystemTempC)
	s.cpuUsage = target.CPUUsagePercent
	s.gpuUsage = float64(target.GPUUsagePercent)
	s.ramUsage = target.RAMUsagePercent
	s.cpuSamples = []float64{target.CPUUsagePercent}
	s.gpuSamples = []float64{float64(target.GPUUsagePercent)}
	s.cpuFan = float64(target.CPUFanRPM)
	s.gpuFan = float64(target.GPUFanRPM)
	s.cpuFreq = float64(target.CPUFrequencyMHz)
	s.gpuFreq = float64(target.GPUFrequencyMHz)
	s.wifiDown = target.WIFIDownloadKbps
	s.wifiUp = target.WIFIUploadKbps
	s.ethDown = target.EthernetDownloadKbps
	s.ethUp = target.EthernetUploadKbps
}

// Step 执行一步平滑阻尼插值。
//
// alpha 在 (0, 1] 之间。若有数据正在过渡中，返回 true；当所有数值均已收敛至目标值，返回 false。
func (s *Smoother) Step(alpha float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return false
	}
	if alpha <= 0 || alpha > 1 {
		alpha = DefaultSmoothAlpha
	}

	changed := false
	changed = lerp(&s.cpuTemp, float64(s.target.CPUTemperatureC), alpha, 0.05) || changed
	changed = lerp(&s.gpuTemp, float64(s.target.GPUTemperatureC), alpha, 0.05) || changed
	changed = lerp(&s.boardTemp, float64(s.target.SystemTempC), alpha, 0.05) || changed

	// 针对 CPU / GPU 负载设置 0.15% 的死区阈值，微幅无意义晃动不触发重绘，锁定稳定读数
	changed = lerp(&s.cpuUsage, s.target.CPUUsagePercent, alpha, 0.15) || changed
	changed = lerp(&s.gpuUsage, float64(s.target.GPUUsagePercent), alpha, 0.15) || changed
	changed = lerp(&s.ramUsage, s.target.RAMUsagePercent, alpha, 0.05) || changed

	changed = lerp(&s.cpuFan, float64(s.target.CPUFanRPM), alpha, 0.8) || changed
	changed = lerp(&s.gpuFan, float64(s.target.GPUFanRPM), alpha, 0.8) || changed

	changed = lerp(&s.cpuFreq, float64(s.target.CPUFrequencyMHz), alpha, 1.0) || changed
	changed = lerp(&s.gpuFreq, float64(s.target.GPUFrequencyMHz), alpha, 1.0) || changed

	changed = lerp(&s.wifiDown, s.target.WIFIDownloadKbps, alpha, 0.2) || changed
	changed = lerp(&s.wifiUp, s.target.WIFIUploadKbps, alpha, 0.2) || changed
	changed = lerp(&s.ethDown, s.target.EthernetDownloadKbps, alpha, 0.2) || changed
	changed = lerp(&s.ethUp, s.target.EthernetUploadKbps, alpha, 0.2) || changed

	// 插值计算后再次钳制，坚决不允许浮点下溢产生负数
	s.cpuTemp = math.Max(0, s.cpuTemp)
	s.gpuTemp = math.Max(0, s.gpuTemp)
	s.boardTemp = math.Max(0, s.boardTemp)
	s.cpuUsage = math.Max(0, math.Min(100, s.cpuUsage))
	s.gpuUsage = math.Max(0, math.Min(100, s.gpuUsage))
	s.ramUsage = math.Max(0, math.Min(100, s.ramUsage))
	s.cpuFan = math.Max(0, s.cpuFan)
	s.gpuFan = math.Max(0, s.gpuFan)
	s.cpuFreq = math.Max(0, s.cpuFreq)
	s.gpuFreq = math.Max(0, s.gpuFreq)
	s.wifiDown = math.Max(0, s.wifiDown)
	s.wifiUp = math.Max(0, s.wifiUp)
	s.ethDown = math.Max(0, s.ethDown)
	s.ethUp = math.Max(0, s.ethUp)

	return changed
}

// Current 返回当前平滑帧的遥测数据以及指标列表。
func (s *Smoother) Current() (domain.MonitorData, []Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := s.target
	data.CPUTemperatureC = int(math.Round(math.Max(0, s.cpuTemp)))
	data.GPUTemperatureC = int(math.Round(math.Max(0, s.gpuTemp)))
	data.SystemTempC = int(math.Round(math.Max(0, s.boardTemp)))
	data.CPUUsagePercent = math.Max(0, math.Min(100, s.cpuUsage))
	data.GPUUsagePercent = int(math.Round(math.Max(0, math.Min(100, s.gpuUsage))))
	data.RAMUsagePercent = math.Max(0, math.Min(100, s.ramUsage))
	data.CPUFanRPM = int(math.Round(math.Max(0, s.cpuFan)))
	data.GPUFanRPM = int(math.Round(math.Max(0, s.gpuFan)))
	data.CPUFrequencyMHz = int(math.Round(math.Max(0, s.cpuFreq)))
	data.GPUFrequencyMHz = int(math.Round(math.Max(0, s.gpuFreq)))
	data.WIFIDownloadKbps = math.Max(0, s.wifiDown)
	data.WIFIUploadKbps = math.Max(0, s.wifiUp)
	data.EthernetDownloadKbps = math.Max(0, s.ethDown)
	data.EthernetUploadKbps = math.Max(0, s.ethUp)

	metrics := []Metric{
		{
			Label:   "CPU 温度",
			Value:   FormatCelsius(data.CPUTemperatureC),
			Number:  s.cpuTemp,
			Detail:  strings.Join([]string{FormatPercent(data.CPUUsagePercent), FormatMHz(data.CPUFrequencyMHz)}, " · "),
			Percent: clamp01(data.CPUUsagePercent / 100),
			Level:   TemperatureLevel(data.CPUTemperatureC),
		},
		{
			Label:   "GPU 温度",
			Value:   FormatCelsius(data.GPUTemperatureC),
			Number:  s.gpuTemp,
			Detail:  strings.Join([]string{FormatPercent(float64(data.GPUUsagePercent)), FormatMHz(data.GPUFrequencyMHz)}, " · "),
			Percent: clamp01(float64(data.GPUUsagePercent) / 100),
			Level:   TemperatureLevel(data.GPUTemperatureC),
		},
		{
			Label:   "内存占用",
			Value:   FormatPercent(data.RAMUsagePercent),
			Number:  data.RAMUsagePercent,
			Detail:  FormatMB(data.RAMTotalMB) + " @ " + FormatMemoryClock(data.RAMFrequencyMHz),
			Percent: clamp01(data.RAMUsagePercent / 100),
			Level:   LoadLevel(data.RAMUsagePercent),
		},
		{
			Label:   "CPU 风扇",
			Value:   FormatRPM(data.CPUFanRPM),
			Number:  s.cpuFan,
			Detail:  FormatMHz(data.CPUFrequencyMHz),
			Percent: 0,
			Level:   LevelNormal,
		},
		{
			Label:   "GPU 风扇",
			Value:   FormatRPM(data.GPUFanRPM),
			Number:  s.gpuFan,
			Detail:  FormatMHz(data.GPUFrequencyMHz),
			Percent: 0,
			Level:   LevelNormal,
		},
		{
			Label:   "主板温度",
			Value:   FormatCelsius(data.SystemTempC),
			Number:  s.boardTemp,
			Detail:  "供电与主板",
			Percent: 0,
			Level:   TemperatureLevel(data.SystemTempC),
		},
	}

	return data, metrics
}

// cleanMonitor 清洗过滤硬件遥测中的异常负数（例如独显休眠返回 -1、网卡离线返回 -1 等）。
func cleanMonitor(raw domain.MonitorData) domain.MonitorData {
	clean := raw

	// 占用率严格限制在 0 到 100 之间
	clean.CPUUsagePercent = clampFloat(clean.CPUUsagePercent, 0, 100)
	if clean.GPUUsagePercent < 0 {
		clean.GPUUsagePercent = 0
	} else if clean.GPUUsagePercent > 100 {
		clean.GPUUsagePercent = 100
	}
	clean.RAMUsagePercent = clampFloat(clean.RAMUsagePercent, 0, 100)

	// 温度、转速、频率与网络速率必须为非负物理量
	clean.CPUTemperatureC = maxInt(0, clean.CPUTemperatureC)
	clean.GPUTemperatureC = maxInt(0, clean.GPUTemperatureC)
	clean.SystemTempC = maxInt(0, clean.SystemTempC)

	clean.CPUFanRPM = maxInt(0, clean.CPUFanRPM)
	clean.GPUFanRPM = maxInt(0, clean.GPUFanRPM)

	clean.CPUFrequencyMHz = maxInt(0, clean.CPUFrequencyMHz)
	clean.GPUFrequencyMHz = maxInt(0, clean.GPUFrequencyMHz)

	clean.WIFIDownloadKbps = math.Max(0, clean.WIFIDownloadKbps)
	clean.WIFIUploadKbps = math.Max(0, clean.WIFIUploadKbps)
	clean.EthernetDownloadKbps = math.Max(0, clean.EthernetDownloadKbps)
	clean.EthernetUploadKbps = math.Max(0, clean.EthernetUploadKbps)

	return clean
}

// smoothNetworkRate 对网络突发流量进行低通滤波：上升积极响应，下降平缓衰减，杜绝数字闪烁。
func smoothNetworkRate(oldRate, newRate float64) float64 {
	if newRate > oldRate {
		// 上升通道：80% 权重快速体现当前正在传输
		return oldRate*0.20 + newRate*0.80
	}
	// 下降通道：平缓淡出衰减，避免瞬时没发包就直接掉到 0 剧烈抖动
	return oldRate*0.45 + newRate*0.55
}

func clampFloat(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// weightedAverage 计算滑动窗口的加权平均，越新的采样权重大，越旧的采样提供稳定性支撑。
func weightedAverage(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	var weightSum float64
	for i, v := range samples {
		w := float64(i + 1)
		sum += v * w
		weightSum += w
	}
	return sum / weightSum
}

// lerp 执行一次单值插值，若差值在误差范围内则直接到位。
func lerp(cur *float64, target float64, alpha float64, eps float64) bool {
	diff := target - *cur
	if math.Abs(diff) <= eps {
		if *cur != target {
			*cur = target
			return true
		}
		return false
	}
	*cur += diff * alpha
	return true
}
