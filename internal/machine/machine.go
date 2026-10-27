// Package machine reaches the machine through the services of the original
// software.
//
// The original AcerAgentService is the main road. It owns the keyboard
// lighting, the fans, the performance profiles, the graphics switch and the
// readings of the machine, and it runs as the system, so the program needs
// no rights of an administrator to use it.
//
// The settings the service does not know are asked from the ACPI provider:
// the boost of the fans, the table of the fans, the overclocking ability and
// the raw buffer of the backlight. That road needs the rights of an
// administrator and it is opened on first use, so a machine without the
// rights still gets everything the service knows.
//
// 实测结论：原厂服务的风扇参数有三个模式，0 是自动，1 是全速，2 是固定
// 占空比。全速时风扇从 2722 RPM 升到 7092 RPM，固定占空比时占空比写在每个
// 风扇的 fan_custom_speed 里，fan_custom_auto 为零。
package machine

import (
	"errors"
	"fmt"
	"time"

	"viper/internal/agentservice"
	"viper/internal/hardware"
)

// ErrNoProvider says that a setting needs the ACPI provider and that the
// provider cannot be reached.
var ErrNoProvider = errors.New("machine: 这一项需要固件接口，原厂服务不提供，需要管理员权限")

// Service is the set of abilities of the AcerAgentService that the program
// uses.
//
// The interface keeps the program testable: a test can hand the program a
// stand for the service, so that the behaviour can be checked without the
// service of the original software.
type Service interface {
	// Get asks for the value of a setting.
	Get(function string) (agentservice.Answer, error)
	// Set writes a setting.
	Set(function string, parameter any) (agentservice.Answer, error)
	// MonitorData asks for the live numbers of the machine.
	MonitorData() (agentservice.Answer, error)
	// Capability asks for an ability of the machine.
	Capability(function string) (agentservice.Answer, error)
	// Close ends the session.
	Close() error
}

// Options change the behaviour of a backend.
type Options struct {
	// Address is the address of the AcerAgentService. The default is the
	// local port of the service.
	Address string
	// Timeout limits the time of a single request.
	Timeout time.Duration
}

// Backend reaches the machine.
//
// The service of the original software answers everything: the readings, the
// settings, the lighting. There is no second road.
type Backend struct {
	service Service
}

// New opens the connection to the AcerAgentService.
//
// The service has to be running. A machine without it is answered with an
// error that says so, the program does not fall back to the firmware on its
// own, because the caller has to know which road it is on.
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

// NewWithService opens the program with the given session of the service.
//
// It is used by the tests, so that they do not need the service of the
// original software.
func NewWithService(service Service) *Backend {
	return &Backend{service: service}
}

// The backend reaches the whole of the machine.
var _ hardware.Gaming = (*Backend)(nil)

// Instance returns the name of the service that holds the machine.
func (b *Backend) Instance() string { return "AcerAgentService" }

// Close releases the connection.
func (b *Backend) Close() {
	if b.service != nil {
		b.service.Close()
	}
}

// get asks the service for the value of a setting.
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

// set writes a setting.
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

// Sensors reads the current temperatures and fan speeds.
//
// 原厂服务的监视数据里有处理器温度、处理器风扇转速、主板温度、显卡温度与
// 显卡风扇转速，它们直接来自机器。
func (b *Backend) Sensors() (hardware.Sensors, error) {
	answer, err := b.service.MonitorData()
	if err != nil {
		return hardware.Sensors{}, fmt.Errorf("machine: 读取监视数据失败: %w", err)
	}
	if err, failed := answer.Err(); failed {
		return hardware.Sensors{}, err
	}

	var data struct {
		CPUTemperature float64 `json:"CPU_TEMPERATURE"`
		CPUFanSpeed    float64 `json:"CPU_FANSPEED"`
		SystemTemp     float64 `json:"SYS1_TEMPERATURE"`
		GPUFanSpeed    float64 `json:"GPU1_FANSPEED"`
		GPUTemp        float64 `json:"GPU1_TEMPERATURE"`
	}
	if err := answer.Decode(&data); err != nil {
		return hardware.Sensors{}, err
	}

	return hardware.Sensors{
		CPUTempC:    int(data.CPUTemperature),
		CPUFanRPM:   int(data.CPUFanSpeed),
		GPUTempC:    int(data.GPUTemp),
		GPUFanRPM:   int(data.GPUFanSpeed),
		SystemTempC: int(data.SystemTemp),
	}, nil
}

// PowerMode returns the performance profile.
func (b *Backend) PowerMode() (hardware.PowerMode, error) {
	answer, err := b.get(agentservice.FunctionOperatingMode)
	if err != nil {
		return hardware.PowerUnknown, err
	}

	var data struct {
		Mode int `json:"mode"`
	}
	if err := answer.Decode(&data); err != nil {
		return hardware.PowerUnknown, err
	}
	return hardware.DecodePowerMode(uint64(data.Mode)), nil
}

// SetPowerMode applies a performance profile.
//
// 原厂服务用的编号是 0 静音、1 均衡、4 性能、5 加速、6 节能，和固件那一侧
// 的编号一样，所以两边对同一个数字的理解不会差开。
//
// 写完读回来核对：机器没有换档，就在错误信息里说清楚。
func (b *Backend) SetPowerMode(mode hardware.PowerMode) error {
	code := hardware.PowerModeCode(mode)
	if err := b.set(agentservice.FunctionOperatingMode, map[string]any{
		"mode": int(code),
	}); err != nil {
		return err
	}
	return b.verifyPowerMode(int(code))
}

// verifyPowerMode reads the performance profile back.
//
// 性能档位的载荷里是 mode，不是开关那几个设置共用的 status，所以它自己核对。
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

// GPUMode returns the graphics switch of the machine.
//
// 原厂服务从零开始数，Viper 从一开始数，所以这里的编号要差一。
func (b *Backend) GPUMode() (hardware.GPUMode, error) {
	answer, err := b.get(agentservice.FunctionGPUMode)
	if err != nil {
		return hardware.GPUModeHybrid, err
	}

	var data struct {
		Mode int `json:"mode"`
	}
	if err := answer.Decode(&data); err != nil {
		return hardware.GPUModeHybrid, err
	}
	return serviceGPUMode(data.Mode), nil
}

// SetGPUMode selects the graphics switch of the machine.
//
// 写入编号 0 是混合、1 是仅独显、2 是自动。切换图形模式要重启才生效，这与
// 官方应用的行为一致。
//
// 这一项**不读回核对**：图形模式写进的是杂项设置表，机器要到重启才真的
// 换过去，服务的结果码就是「收到了」的凭据。读回来的值在这一项上说明不了
// 什么。
func (b *Backend) SetGPUMode(mode hardware.GPUMode) error {
	code := int(mode)
	if code < 1 {
		code = 1
	}
	return b.set(agentservice.FunctionGPUMode, map[string]any{
		"mode": code - 1,
	})
}

// serviceGPUMode maps the number of the service onto a graphics mode.
func serviceGPUMode(value int) hardware.GPUMode {
	switch value {
	case 0:
		return hardware.GPUModeHybrid
	case 1:
		return hardware.GPUModeDiscrete
	case 2:
		return hardware.GPUModeAuto
	default:
		return hardware.GPUModeHybrid
	}
}

// DiscreteGPU reports the state of the discrete graphics switch.
//
// 原厂服务报告图形模式，独显模式就是独显已经接管，所以这一项不用问固件。
func (b *Backend) DiscreteGPU() (bool, error) {
	mode, err := b.GPUMode()
	if err != nil {
		return false, err
	}
	return mode == hardware.GPUModeDiscrete, nil
}

// OverclockCapability reports which overclocking features the machine
// supports.
//
// 原厂服务自己也不实现这一项，它的应答是失败码 3，所以这一项问固件。
// OverclockCapability reports which overclocking features the machine supports.
//
// 原厂服务自己也不实现这一项（结果码 3），所以这一项报成不支持。
func (b *Backend) OverclockCapability() (gpu bool, cpu bool, err error) {
	answer, err := b.service.Capability(agentservice.CapabilityOverclockingMode)
	if err != nil {
		return false, false, fmt.Errorf("machine: 读取超频能力失败: %w", err)
	}
	if err, failed := answer.Err(); failed {
		// 服务不实现这一项：它报的是"不支持"，不是出错。
		if errors.Is(err, agentservice.ErrUnsupported) {
			return false, false, nil
		}
		return false, false, err
	}
	return false, false, nil
}

// WinKeyLocked reports whether the Windows key is locked.
//
// 实测：原厂服务的 WIN_KEY **写不动**——它收下请求（结果码 0），但读回来的
// 值一直是一个，写 0 写 1 都一样。所以这一项走固件那一侧（锁是固件档案里的
// 一个位），那一边需要管理员权限。
func (b *Backend) WinKeyLocked() (bool, error) {
	status, err := b.switchStatus(agentservice.FunctionWinKey)
	if err != nil {
		return false, err
	}
	return status == 0, nil
}

// SetWinKeyLocked locks or unlocks the Windows key.
//
// 这一项走固件那一侧，理由见 WinKeyLocked。
func (b *Backend) SetWinKeyLocked(locked bool) error {
	status := 1
	if locked {
		status = 0
	}
	if err := b.set(agentservice.FunctionWinKey, map[string]any{"status": status}); err != nil {
		return err
	}
	// 写完要读回核对：实测这台机器上服务收下请求却不生效，核对会把它报出来。
	return b.verifyStatus(agentservice.FunctionWinKey, status)
}

// LCDOverdrive reports whether the display overdrive is active.
func (b *Backend) LCDOverdrive() (bool, error) {
	status, err := b.switchStatus(agentservice.FunctionLCDOverdrive)
	if err != nil {
		return false, err
	}
	return status == 1, nil
}

// SetLCDOverdrive enables or disables the display overdrive.
func (b *Backend) SetLCDOverdrive(enabled bool) error {
	status := boolStatus(enabled)
	if err := b.set(agentservice.FunctionLCDOverdrive, map[string]any{"status": status}); err != nil {
		return err
	}
	return b.verifyStatus(agentservice.FunctionLCDOverdrive, status)
}

// BootSound reports whether the sound at the start of the machine is on.
func (b *Backend) BootSound() (bool, error) {
	status, err := b.switchStatus(agentservice.FunctionBootSound)
	if err != nil {
		return false, err
	}
	return status == 1, nil
}

// SetBootSound switches the sound at the start of the machine on or off.
func (b *Backend) SetBootSound(enabled bool) error {
	status := boolStatus(enabled)
	if err := b.set(agentservice.FunctionBootSound, map[string]any{"status": status}); err != nil {
		return err
	}
	return b.verifyStatus(agentservice.FunctionBootSound, status)
}

// verifyStatus reads a switch back and compares it with what was written.
//
// 原厂服务把写下的状态原样报回来，所以读回来的不一样，就是机器没有接受。
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

// poll calls a reading until it agrees with what was written.
//
// 有的设置不是立刻生效的：服务把它交给别的服务或者写进固件，读回来的值会
// 晚一点才变。所以读回核对要等一小会儿，而不是读一次就下结论。
func poll(read func() (bool, error)) (bool, error) {
	deadline := time.Now().Add(verifyTimeout)
	for {
		ok, err := read()
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		time.Sleep(verifyInterval)
	}
}

// The times of the read back check.
//
// 写进机器的设置有的要过别的服务才生效，读回来会晚一点。这两个值是可变的，
// 测试会把等待的时间改短。
var (
	verifyTimeout  = 3 * time.Second
	verifyInterval = 100 * time.Millisecond
)

// switchStatus reads the state of a switch.
//
// 原厂服务的开关类设置都用同一个形状：状态是零或者一。
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

// boolStatus turns a switch into the number the service uses.
func boolStatus(enabled bool) int {
	if enabled {
		return 1
	}
	return 0
}
