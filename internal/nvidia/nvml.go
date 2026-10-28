//go:build windows

// Package nvidia 提供基于 NVIDIA NVML (nvml.dll) 的显卡遥测数据采集功能。
package nvidia

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

const (
	clockGraphics = 0
	clockMemory   = 2
)

type utilization struct {
	GPU    uint32
	Memory uint32
}

type memory struct {
	Total uint64
	Free  uint64
	Used  uint64
}

type memoryV2 struct {
	Version uint64
	Total   uint64
	Free    uint64
	Used    uint64
}

type entry func(arguments ...uintptr) (result uintptr, present bool)

type library struct {
	handle syscall.Handle

	init                entry
	shutdown            entry
	deviceCount         entry
	deviceHandle        entry
	deviceUtilization   entry
	deviceClock         entry
	deviceMaxClock      entry
	deviceName          entry
	deviceMemory        entry
	deviceMemoryV2      entry
	devicePState        entry
	systemDriverVersion entry
}

func wrap(dll *syscall.LazyDLL, name string) entry {
	proc := dll.NewProc(name)
	return func(arguments ...uintptr) (uintptr, bool) {
		if err := proc.Find(); err != nil {
			return 0, false
		}
		result, _, _ := proc.Call(arguments...)
		return result, true
	}
}

func call(proc entry, arguments ...uintptr) (uintptr, bool) {
	if proc == nil {
		return 0, false
	}
	return proc(arguments...)
}

// Telemetry 记录单个 NVIDIA GPU 的硬件遥测指标。
type Telemetry struct {
	// Name 显卡型号名称。
	Name string

	// UsagePercent 核心利用率百分比。
	UsagePercent int

	// MemoryUsagePercent 显存控制器利用率百分比。
	MemoryUsagePercent int

	// GraphicsClockMHz 当前核心主频（MHz）。
	GraphicsClockMHz int

	// MemoryClockMHz 当前显存工作频率（MHz）。
	MemoryClockMHz int

	// MaxGraphicsClockMHz 显卡核心最大加速频率（MHz）。
	MaxGraphicsClockMHz int

	// MemoryUsedBytes 已使用的显存大小（字节）。
	MemoryUsedBytes uint64

	// MemoryTotalBytes 显存总容量（字节）。
	MemoryTotalBytes uint64

	// PState 性能电源状态等级（P-State）。
	PState int

	// Idle 标识显卡是否处于空闲省电状态。
	Idle bool
}

// IsIdle 判断显卡核心利用率是否为 0。
func (t Telemetry) IsIdle() bool {
	return t.UsagePercent == 0
}

var (
	mutex     sync.Mutex
	loaded    bool
	loadError error
	current   *library
)

func load() (*library, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if loaded {
		return current, loadError
	}

	handle, err := syscall.LoadLibrary("nvml.dll")
	if err != nil {
		loadError = fmt.Errorf("nvidia: the management library of the driver is not available: %w", err)
		current = nil
		loaded = true
		return nil, loadError
	}

	dll := syscall.NewLazyDLL("nvml.dll")
	current = &library{
		handle:              handle,
		init:                wrap(dll, "nvmlInit_v2"),
		shutdown:            wrap(dll, "nvmlShutdown"),
		deviceCount:         wrap(dll, "nvmlDeviceGetCount_v2"),
		deviceHandle:        wrap(dll, "nvmlDeviceGetHandleByIndex_v2"),
		deviceUtilization:   wrap(dll, "nvmlDeviceGetUtilizationRates"),
		deviceClock:         wrap(dll, "nvmlDeviceGetClockInfo"),
		deviceMaxClock:      wrap(dll, "nvmlDeviceGetMaxClockInfo"),
		deviceName:          wrap(dll, "nvmlDeviceGetName"),
		deviceMemory:        wrap(dll, "nvmlDeviceGetMemoryInfo"),
		deviceMemoryV2:      wrap(dll, "nvmlDeviceGetMemoryInfo_v2"),
		devicePState:        wrap(dll, "nvmlDeviceGetPState"),
		systemDriverVersion: wrap(dll, "nvmlSystemGetDriverVersion"),
	}
	loadError = nil
	loaded = true
	return current, nil
}

// Available 检查系统是否安装并可正常加载 NVML 动态链接库。
func Available() bool {
	_, err := load()
	return err == nil
}

// Close 关闭 NVML 库并释放相关资源。
func Close() {
	mutex.Lock()
	defer mutex.Unlock()

	if current == nil || current.handle == 0 {
		return
	}
	current.shutdown()
	syscall.FreeLibrary(current.handle)
	current = nil
	loadError = nil
	loaded = false
}

// Read 查询并返回系统上所有 NVIDIA 显卡的遥测快照。
func Read() ([]Telemetry, error) {
	lib, err := load()
	if err != nil {
		return nil, err
	}

	result, _ := lib.init()
	if int32(result) != 0 {
		return nil, fmt.Errorf("nvidia: the driver refused the access, code %d", int32(result))
	}

	var count uint32
	result, _ = lib.deviceCount(uintptr(unsafe.Pointer(&count)))
	if int32(result) != 0 {
		return nil, fmt.Errorf("nvidia: the cards cannot be counted, code %d", int32(result))
	}

	out := make([]Telemetry, 0, count)
	for index := uint32(0); index < count; index++ {
		var handle uintptr
		result, _ = lib.deviceHandle(
			uintptr(index),
			uintptr(unsafe.Pointer(&handle)),
		)
		if int32(result) != 0 {
			continue
		}
		out = append(out, read(lib, handle))
	}
	return out, nil
}

func read(lib *library, handle uintptr) Telemetry {
	telemetry := Telemetry{PState: -1}

	if name, ok := readName(lib, handle); ok {
		telemetry.Name = name
	}

	var load utilization
	if result, ok := call(lib.deviceUtilization,
		handle,
		uintptr(unsafe.Pointer(&load)),
	); ok && int32(result) == 0 {
		telemetry.UsagePercent = int(load.GPU)
		telemetry.MemoryUsagePercent = int(load.Memory)
	}

	var clock uint32
	if result, ok := call(lib.deviceClock,
		handle,
		uintptr(clockGraphics),
		uintptr(unsafe.Pointer(&clock)),
	); ok && int32(result) == 0 {
		telemetry.GraphicsClockMHz = int(clock)
	}

	if result, ok := call(lib.deviceClock,
		handle,
		uintptr(clockMemory),
		uintptr(unsafe.Pointer(&clock)),
	); ok && int32(result) == 0 {
		telemetry.MemoryClockMHz = int(clock)
	}

	if result, ok := call(lib.deviceMaxClock,
		handle,
		uintptr(clockGraphics),
		uintptr(unsafe.Pointer(&clock)),
	); ok && int32(result) == 0 {
		telemetry.MaxGraphicsClockMHz = int(clock)
	}

	var info memory
	if result, ok := call(lib.deviceMemory,
		handle,
		uintptr(unsafe.Pointer(&info)),
	); ok && int32(result) == 0 {
		telemetry.MemoryUsedBytes = info.Used
		telemetry.MemoryTotalBytes = info.Total
	} else {
		var infoV2 memoryV2
		if result, ok := call(lib.deviceMemoryV2,
			handle,
			uintptr(unsafe.Pointer(&infoV2)),
		); ok && int32(result) == 0 {
			telemetry.MemoryUsedBytes = infoV2.Used
			telemetry.MemoryTotalBytes = infoV2.Total
		}
	}

	var state uint32
	if result, ok := call(lib.devicePState,
		handle,
		uintptr(unsafe.Pointer(&state)),
	); ok && int32(result) == 0 {
		telemetry.PState = int(state)
	}

	return telemetry
}

func readName(lib *library, handle uintptr) (string, bool) {
	buffer := make([]byte, 96)
	result, ok := call(lib.deviceName,
		handle,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if !ok || int32(result) != 0 {
		return "", false
	}
	for i, b := range buffer {
		if b == 0 {
			return string(buffer[:i]), true
		}
	}
	return string(buffer), true
}

// DriverVersion 查询当前安装的 NVIDIA 显卡驱动版本字符串。
func DriverVersion() (string, error) {
	lib, err := load()
	if err != nil {
		return "", err
	}
	buffer := make([]byte, 96)
	result, ok := call(lib.systemDriverVersion,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if !ok {
		return "", fmt.Errorf("nvidia: the driver does not report its version")
	}
	if int32(result) != 0 {
		return "", fmt.Errorf("nvidia: the version of the driver cannot be read, code %d", int32(result))
	}
	for i, b := range buffer {
		if b == 0 {
			return string(buffer[:i]), nil
		}
	}
	return string(buffer), nil
}
