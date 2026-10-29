// Package machine 基于 AcerAgentService 实现 domain.Gaming 接口。
package machine

import (
	"errors"
	"fmt"
	"time"

	"viper/internal/agentservice"
	"viper/internal/domain"
)

// ErrNoProvider 表示目标功能缺少底层固件或服务驱动支持。
var ErrNoProvider = errors.New("machine: 这一项需要固件接口，原厂服务不提供，需要管理员权限")

// Service 定义与原厂服务交互的通信接口抽象。
type Service interface {
	Get(function string) (agentservice.Answer, error)
	Set(function string, parameter any) (agentservice.Answer, error)
	MonitorData() (agentservice.Answer, error)
	Capability(function string) (agentservice.Answer, error)
	Close() error
}

// Options 定义初始化机器控制后端的选项。
type Options struct {
	Address string
	Timeout time.Duration
}

// Backend 封装针对 Acer 笔记本硬件的具体控制逻辑。
type Backend struct {
	service Service
}

// New 创建并连接至 AcerAgentService 的硬件控制后端。
func New(options Options) (*Backend, error) {
	client, err := agentservice.Connect(agentservice.Options{
		Address: options.Address,
		Timeout: options.Timeout,
	})
	if err != nil {
		return nil, err
	}

	return &Backend{service: client}, nil
}

// NewWithService 使用已初始化的 Service 实例构建后端。
func NewWithService(service Service) *Backend {
	return &Backend{service: service}
}

var _ domain.Gaming = (*Backend)(nil)

// Instance 返回当前后端的标识名称。
func (b *Backend) Instance() string { return "AcerAgentService" }

// Close 关闭后端并释放底层服务连接。
func (b *Backend) Close() {
	if b.service != nil {
		b.service.Close()
	}
}

func (b *Backend) get(function string) (agentservice.Answer, error) {
	answer, err := b.service.Get(function)
	if err != nil {
		return agentservice.Answer{}, fmt.Errorf("machine: 读取 %s 失败: %w", function, err)
	}
	if err, failed := answer.Err(); failed {
		return agentservice.Answer{}, err
	}
	return answer, nil
}

func (b *Backend) set(function string, parameter any) error {
	answer, err := b.service.Set(function, parameter)
	if err != nil {
		return fmt.Errorf("machine: 写入 %s 失败: %w", function, err)
	}
	if err, failed := answer.Err(); failed {
		return err
	}
	return nil
}

// Monitor 获取全套硬件运行状态与指标监控数据。
func (b *Backend) Monitor() (domain.MonitorData, error) {
	answer, err := b.service.MonitorData()
	if err != nil {
		return domain.MonitorData{}, fmt.Errorf("machine: 读取监视数据失败: %w", err)
	}
	if err, failed := answer.Err(); failed {
		return domain.MonitorData{}, err
	}

	var data struct {
		CPUTemperature   float64   `json:"CPU_TEMPERATURE"`
		CPUFanSpeed      float64   `json:"CPU_FANSPEED"`
		SystemTemp       float64   `json:"SYS1_TEMPERATURE"`
		GPUTemperature   float64   `json:"GPU1_TEMPERATURE"`
		GPUFanSpeed      float64   `json:"GPU1_FANSPEED"`
		GPUIdle          float64   `json:"GPU1_IDLE_STATUS"`
		GPUUsage         float64   `json:"GPU1_USAGE"`
		GPUFrequency     float64   `json:"GPU1_FREQUENCY"`
		GPUMaxFrequency  float64   `json:"GPU1_MAX_FREQUENCY"`
		CPUFrequency     float64   `json:"CPU_FREQUENCY"`
		CPUMaxFrequency  float64   `json:"CPU_MAX_FREQUENCY"`
		CPUUsage         float64   `json:"CPU_USAGE"`
		CPUCoreFrequency []float64 `json:"CPU_CORE_FREQUENCY"`
		RAMUsage         float64   `json:"RAM_USAGE"`
		RAMTotal         float64   `json:"RAM_TOTAL"`
		RAMFrequency     float64   `json:"RAM_FREQUENCY"`
		WIFIDownload     float64   `json:"WIFI_DOWNLOAD"`
		WIFIUpload       float64   `json:"WIFI_UPLOAD"`
		EtherDownload    float64   `json:"ETHERNET_DOWNLOAD"`
		EtherUpload      float64   `json:"ETHERNET_UPLOAD"`
	}
	if err := answer.Decode(&data); err != nil {
		return domain.MonitorData{}, err
	}

	gpuUsage := int(data.GPUUsage)
	if gpuUsage < 0 {
		gpuUsage = 0
	}
	cpuUsage := data.CPUUsage
	if cpuUsage < 0 {
		cpuUsage = 0
	}
	ramUsage := data.RAMUsage
	if ramUsage < 0 {
		ramUsage = 0
	}

	result := domain.MonitorData{
		CPUTemperatureC:      maxInt(0, int(data.CPUTemperature)),
		CPUFanRPM:            maxInt(0, int(data.CPUFanSpeed)),
		CPUUsagePercent:      cpuUsage,
		CPUFrequencyMHz:      maxInt(0, int(data.CPUFrequency)),
		CPUMaxFrequencyMHz:   maxInt(0, int(data.CPUMaxFrequency)),
		SystemTempC:          maxInt(0, int(data.SystemTemp)),
		GPUTemperatureC:      maxInt(0, int(data.GPUTemperature)),
		GPUFanRPM:            maxInt(0, int(data.GPUFanSpeed)),
		GPUUsagePercent:      gpuUsage,
		GPUFrequencyMHz:      maxInt(0, int(data.GPUFrequency)),
		GPUMaxFrequencyMHz:   maxInt(0, int(data.GPUMaxFrequency)),
		GPUIdle:              data.GPUIdle != 0,
		RAMTotalMB:           maxFloat(0, data.RAMTotal),
		RAMUsagePercent:      ramUsage,
		RAMFrequencyMHz:      maxInt(0, int(data.RAMFrequency)),
		WIFIDownloadKbps:     maxFloat(0, data.WIFIDownload),
		WIFIUploadKbps:       maxFloat(0, data.WIFIUpload),
		EthernetDownloadKbps: maxFloat(0, data.EtherDownload),
		EthernetUploadKbps:   maxFloat(0, data.EtherUpload),
	}
	for _, clock := range data.CPUCoreFrequency {
		result.CPUCoreFrequencyMHz = append(result.CPUCoreFrequencyMHz, int(clock))
	}
	return result, nil
}

// Sensors 读取基本的 CPU/GPU 温度及风扇转速。
func (b *Backend) Sensors() (domain.Sensors, error) {
	data, err := b.Monitor()
	if err != nil {
		return domain.Sensors{}, err
	}

	return domain.Sensors{
		CPUTempC:    data.CPUTemperatureC,
		CPUFanRPM:   data.CPUFanRPM,
		GPUTempC:    data.GPUTemperatureC,
		GPUFanRPM:   data.GPUFanRPM,
		SystemTempC: data.SystemTempC,
	}, nil
}

// PowerMode 查询当前电源性能模式。
func (b *Backend) PowerMode() (domain.PowerMode, error) {
	answer, err := b.get(agentservice.FunctionOperatingMode)
	if err != nil {
		return domain.PowerUnknown, err
	}

	var data struct {
		Mode int `json:"mode"`
	}
	if err := answer.Decode(&data); err != nil {
		return domain.PowerUnknown, err
	}
	return domain.DecodePowerMode(uint64(data.Mode)), nil
}

// SetPowerMode 设置并校验目标电源性能模式。
func (b *Backend) SetPowerMode(mode domain.PowerMode) error {
	code := domain.PowerModeCode(mode)
	if err := b.set(agentservice.FunctionOperatingMode, map[string]any{
		"mode": int(code),
	}); err != nil {
		return err
	}
	return b.verifyPowerMode(int(code))
}

func (b *Backend) verifyPowerMode(expected int) error {
	current := -1
	read := func() (bool, error) {
		answer, err := b.get(agentservice.FunctionOperatingMode)
		if err != nil {
			return false, err
		}

		var data struct {
			Mode int `json:"mode"`
		}
		if err := answer.Decode(&data); err != nil {
			return false, err
		}
		current = data.Mode
		return current == expected, nil
	}

	ok, err := poll(read)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("machine: 性能档位没有生效，服务报的是 %d，不是 %d",
			current, expected)
	}
	return nil
}

// GPUMode 查询显卡直连/混合切换模式。
func (b *Backend) GPUMode() (domain.GPUMode, error) {
	answer, err := b.get(agentservice.FunctionGPUMode)
	if err != nil {
		return domain.GPUModeHybrid, err
	}

	var data struct {
		Mode int `json:"mode"`
	}
	if err := answer.Decode(&data); err != nil {
		return domain.GPUModeHybrid, err
	}
	return serviceGPUMode(data.Mode), nil
}

// SetGPUMode 设置显卡工作模式。
func (b *Backend) SetGPUMode(mode domain.GPUMode) error {
	code := int(mode)
	if code < 1 {
		code = 1
	}
	return b.set(agentservice.FunctionGPUMode, map[string]any{
		"mode": code - 1,
	})
}

func serviceGPUMode(value int) domain.GPUMode {
	switch value {
	case 0:
		return domain.GPUModeHybrid
	case 1:
		return domain.GPUModeDiscrete
	case 2:
		return domain.GPUModeAuto
	default:
		return domain.GPUModeHybrid
	}
}

// DiscreteGPU 判断当前是否工作在纯独显直连模式。
func (b *Backend) DiscreteGPU() (bool, error) {
	mode, err := b.GPUMode()
	if err != nil {
		return false, err
	}
	return mode == domain.GPUModeDiscrete, nil
}

// OverclockCapability 探测 GPU 与 CPU 的超频支持能力。
func (b *Backend) OverclockCapability() (gpu bool, cpu bool, err error) {
	answer, err := b.service.Capability(agentservice.CapabilityOverclockingMode)
	if err != nil {
		return false, false, fmt.Errorf("machine: 读取超频能力失败: %w", err)
	}
	if err, failed := answer.Err(); failed {
		if errors.Is(err, agentservice.ErrUnsupported) {
			return false, false, nil
		}
		return false, false, err
	}
	return false, false, nil
}

// WinKeyLocked 查询 Windows 按键锁定状态。
func (b *Backend) WinKeyLocked() (bool, error) {
	status, err := b.switchStatus(agentservice.FunctionWinKey)
	if err != nil {
		return false, err
	}
	return status == 0, nil
}

// SetWinKeyLocked 启用或禁用 Windows 徽标键锁定。
func (b *Backend) SetWinKeyLocked(locked bool) error {
	status := 1
	if locked {
		status = 0
	}
	if err := b.set(agentservice.FunctionWinKey, map[string]any{"status": status}); err != nil {
		return err
	}

	return b.verifyStatus(agentservice.FunctionWinKey, status)
}

// LCDOverdrive 查询屏幕 LCD Overdrive 响应时间超速开关。
func (b *Backend) LCDOverdrive() (bool, error) {
	status, err := b.switchStatus(agentservice.FunctionLCDOverdrive)
	if err != nil {
		return false, err
	}
	return status == 1, nil
}

// SetLCDOverdrive 设置屏幕 LCD Overdrive 开关。
func (b *Backend) SetLCDOverdrive(enabled bool) error {
	status := boolStatus(enabled)
	if err := b.set(agentservice.FunctionLCDOverdrive, map[string]any{"status": status}); err != nil {
		return err
	}
	return b.verifyStatus(agentservice.FunctionLCDOverdrive, status)
}

// BootSound 查询开机启动音效开关状态。
func (b *Backend) BootSound() (bool, error) {
	status, err := b.switchStatus(agentservice.FunctionBootSound)
	if err != nil {
		return false, err
	}
	return status == 1, nil
}

// SetBootSound 设置开机启动音效开关。
func (b *Backend) SetBootSound(enabled bool) error {
	status := boolStatus(enabled)
	if err := b.set(agentservice.FunctionBootSound, map[string]any{"status": status}); err != nil {
		return err
	}
	return b.verifyStatus(agentservice.FunctionBootSound, status)
}

func (b *Backend) verifyStatus(function string, expected int) error {
	match := false
	read := func() (bool, error) {
		status, err := b.switchStatus(function)
		if err != nil {
			return false, err
		}
		match = status == expected
		return match, nil
	}

	ok, err := poll(read)
	if err != nil {
		return err
	}
	if !ok {
		status, _ := b.switchStatus(function)
		return fmt.Errorf("machine: %s 没有生效，服务报的是 %d，不是 %d",
			function, status, expected)
	}
	return nil
}

func poll(read func() (bool, error)) (bool, error) {
	deadline := time.Now().Add(verifyTimeout)
	var lastErr error
	for {
		ok, err := read()
		switch {
		case err != nil:
			lastErr = err
		case ok:
			return true, nil
		default:
			lastErr = nil
		}
		if time.Now().After(deadline) {
			return false, lastErr
		}
		time.Sleep(verifyInterval)
	}
}

var (
	verifyTimeout  = 3 * time.Second
	verifyInterval = 100 * time.Millisecond
)

func (b *Backend) switchStatus(function string) (int, error) {
	answer, err := b.get(function)
	if err != nil {
		return 0, err
	}

	var data struct {
		Status int `json:"status"`
	}
	if err := answer.Decode(&data); err != nil {
		return 0, err
	}
	return data.Status, nil
}

func boolStatus(enabled bool) int {
	if enabled {
		return 1
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
