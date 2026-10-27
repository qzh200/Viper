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
	"reflect"
	"sync"
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
	// OpenFirmware opens the connection to the ACPI provider. When it is
	// nil the provider is not used at all, and the settings that need it
	// report that they are out of reach.
	//
	// 这一层不自己造固件的连接，连接由外面给进来，所以硬件层与原厂服务
	// 这一层只通过两个接口说话，谁也不认识对方的实现。
	OpenFirmware func() (hardware.Firmware, error)
}

// Backend reaches the machine.
//
// The service answers the readings and the writes of the settings it owns,
// the provider answers the few settings it does not know.
type Backend struct {
	service Service

	// openFirmware opens the connection to the provider.
	openFirmware func() (hardware.Firmware, error)
	// firmware is the connection to the provider, it is opened on first
	// use.
	firmware hardware.Firmware
	// mutex guards the opening of the provider, so that two calls that
	// arrive at the same time do not open two connections.
	mutex sync.Mutex
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

	return &Backend{
		service:      client,
		openFirmware: options.OpenFirmware,
	}, nil
}

// NewWithService opens the program with the given session of the service.
//
// It is used by the tests, so that they do not need the service of the
// original software.
func NewWithService(service Service, openFirmware func() (hardware.Firmware, error)) *Backend {
	return &Backend{service: service, openFirmware: openFirmware}
}

// The backend reaches the whole of the machine.
var _ hardware.Gaming = (*Backend)(nil)

// Instance returns the name of the service that holds the machine.
func (b *Backend) Instance() string { return "AcerAgentService" }

// Close releases the connections.
func (b *Backend) Close() {
	if b.service != nil {
		b.service.Close()
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.firmware != nil {
		b.firmware.Close()
		b.firmware = nil
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

// provider opens the connection to the ACPI provider on first use.
//
// 一个装着空指针的接口不等于 nil，所以这里要看指针本身，不然第一个需要
// 固件的请求就会把程序打崩。
func (b *Backend) provider() (hardware.Firmware, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.firmware != nil {
		return b.firmware, nil
	}
	if b.openFirmware == nil {
		return nil, ErrNoProvider
	}

	firmware, err := b.openFirmware()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoProvider, err)
	}
	if !usableFirmware(firmware) {
		return nil, ErrNoProvider
	}

	b.firmware = firmware
	return firmware, nil
}

// usableFirmware takes a provider that cannot be used for nothing.
func usableFirmware(firmware hardware.Firmware) bool {
	if firmware == nil {
		return false
	}

	value := reflect.ValueOf(firmware)
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		if value.IsNil() {
			return false
		}
	}
	return true
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
func (b *Backend) SetPowerMode(mode hardware.PowerMode) error {
	code := hardware.PowerModeCode(mode)
	return b.set(agentservice.FunctionOperatingMode, map[string]any{
		"mode": int(code),
	})
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
func (b *Backend) OverclockCapability() (gpu bool, cpu bool, err error) {
	firmware, err := b.provider()
	if err != nil {
		return false, false, err
	}
	return firmware.OverclockCapability()
}

// WinKeyLocked reports whether the Windows key is locked.
//
// 原厂服务报告的是这个键还能不能用，编号一是能用，所以锁住的时候是零。
func (b *Backend) WinKeyLocked() (bool, error) {
	status, err := b.switchStatus(agentservice.FunctionWinKey)
	if err != nil {
		return false, err
	}
	return status == 0, nil
}

// SetWinKeyLocked locks or unlocks the Windows key.
func (b *Backend) SetWinKeyLocked(locked bool) error {
	status := 1
	if locked {
		status = 0
	}
	return b.set(agentservice.FunctionWinKey, map[string]any{"status": status})
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
	return b.set(agentservice.FunctionLCDOverdrive, map[string]any{
		"status": boolStatus(enabled),
	})
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
	return b.set(agentservice.FunctionBootSound, map[string]any{
		"status": boolStatus(enabled),
	})
}

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
