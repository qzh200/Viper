// Package hardware drives the Acer ACPI WMI provider.
//
// The provider exposes two classes: AcerGamingFunction, which offers the
// named getters and setters of the gaming hardware, and APGeAction, the
// generic ACPI PNP0C14 action class used by the keyboard and CoolBoost.
//
// Every method signature here was read from the live MOF, which matters
// because the input types differ per method and some methods take no input
// at all.
package hardware

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"viper/internal/wmi"
)

// Class names of the ACPI WMI provider.
const (
	// ClassGaming exposes the Acer gaming hardware functions.
	ClassGaming = "AcerGamingFunction"

	// ClassAction exposes the generic ACPI PNP0C14 action functions.
	ClassAction = "APGeAction"
)

// Sensor categories of the GetGamingSysInfo method. The query is encoded
// as category<<8 | index.
const (
	sysInfoCPUTemp    = 0x0101
	sysInfoCPUFanRPM  = 0x0201
	sysInfoSystemTemp = 0x0301
	sysInfoGPUFanRPM  = 0x0601
	sysInfoGPUTemp    = 0x0A01
)

// Sensors is a snapshot of the machine temperatures and fan speeds.
type Sensors struct {
	CPUTempC    int
	CPUFanRPM   int
	GPUTempC    int
	GPUFanRPM   int
	SystemTempC int
}

// Provider calls a method of a WMI class.
//
// The interface keeps the backend testable: a test can hand a backend a
// provider that answers from a table instead of from the firmware.
type Provider interface {
	// Call invokes a method of a class and returns its output parameters.
	Call(class, method, instancePath string, args map[string]wmi.Value) (map[string]wmi.Value, error)
	// Instances returns the object paths of every instance of a class.
	Instances(class string) ([]string, error)
	// Close releases the connection.
	Close()
}

// Backend talks to the Acer ACPI WMI provider.
//
// 服务的每个连接各有一个 goroutine，它们共用同一个后端。写设置和写完
// 之后的读回校验必须是一整段，不然两个请求会交叉：一个请求写完背光，
// 另一个请求读回时读到的是前一个的值，于是拿到一个假的错误；或者四个
// 分区的颜色来自两份不同的配置，键盘上真的就花了。
type Backend struct {
	client     Provider
	gamingPath string
	actionPath string

	// mutex guards the whole operation on the machine, so that a write
	// and the check that follows it are not interleaved with another
	// operation.
	mutex sync.Mutex
}

// NewBackend connects to the ACPI namespace and resolves the instances that
// carry the gaming functions.
//
// Enumerating the instances requires administrative rights.
func NewBackend() (*Backend, error) {
	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		return nil, err
	}

	backend := &Backend{client: client}

	gamingPaths, err := client.Instances(ClassGaming)
	if err != nil {
		client.Close()
		return nil, err
	}
	if len(gamingPaths) == 0 {
		client.Close()
		return nil, fmt.Errorf("hardware: no %s instance found", ClassGaming)
	}
	backend.gamingPath = gamingPaths[0]

	// APGeAction usually shares the ACPI device with AcerGamingFunction,
	// but the class has to be resolved separately.
	actionPaths, err := client.Instances(ClassAction)
	if err != nil || len(actionPaths) == 0 {
		// Fall back to the instance of the gaming class, which is what the
		// provider exposes on most machines.
		backend.actionPath = backend.gamingPath
	} else {
		backend.actionPath = actionPaths[0]
	}

	return backend, nil
}

// NewBackendWith builds a backend on top of the given provider.
//
// The paths of the instances are used as they are, so that a test can
// tell the calls of the two classes apart.
func NewBackendWith(provider Provider, gamingPath, actionPath string) *Backend {
	if actionPath == "" {
		actionPath = gamingPath
	}
	return &Backend{client: provider, gamingPath: gamingPath, actionPath: actionPath}
}

// Instance returns the resolved object path of the gaming functions.
func (b *Backend) Instance() string { return b.gamingPath }

// operate runs one operation on the machine while holding the lock.
//
// 写设置和写完之后的读回校验是一整段，不能和另一个操作交叉。服务里每个
// 连接各有一个 goroutine，它们共用同一个后端。
//
// 锁只加在这一层，内部的辅助方法不再加锁，所以一个操作里可以调用另一个
// 操作而不会自己等自己。
func operate[T any](b *Backend, body func() (T, error)) (T, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return body()
}

// Close releases the WMI connection.
func (b *Backend) Close() { b.client.Close() }

// callGaming invokes a method of the AcerGamingFunction class.
func (b *Backend) callGaming(method string, args map[string]wmi.Value) (map[string]wmi.Value, error) {
	return b.client.Call(ClassGaming, method, b.gamingPath, args)
}

// callAction invokes a method of the APGeAction class.
func (b *Backend) callAction(method string, args map[string]wmi.Value) (map[string]wmi.Value, error) {
	return b.client.Call(ClassAction, method, b.actionPath, args)
}

// Sensors reads the current temperatures and fan speeds.
func (b *Backend) Sensors() (Sensors, error) {
	var out Sensors

	cpuTemp, err := b.sysInfo(sysInfoCPUTemp, false)
	if err != nil {
		return out, err
	}
	out.CPUTempC = cpuTemp

	cpuFan, err := b.sysInfo(sysInfoCPUFanRPM, true)
	if err != nil {
		return out, err
	}
	out.CPUFanRPM = cpuFan

	sysTemp, err := b.sysInfo(sysInfoSystemTemp, false)
	if err != nil {
		return out, err
	}
	out.SystemTempC = sysTemp

	gpuFan, err := b.sysInfo(sysInfoGPUFanRPM, true)
	if err != nil {
		return out, err
	}
	out.GPUFanRPM = gpuFan

	gpuTemp, err := b.sysInfo(sysInfoGPUTemp, false)
	if err != nil {
		return out, err
	}
	out.GPUTempC = gpuTemp

	return out, nil
}

// sysInfo reads one value through GetGamingSysInfo.
//
// The provider answers with a status in the lowest byte and the reading
// above it. The status has to be zero, otherwise the answer does not
// describe a reading at all.
//
// 转速用整个十六位，温度只用低八位。以前这里靠"高位是不是零"来分辨转速和
// 温度：零度和零转速会落到另一个分支，把状态字节当成读数报出去，而且不看
// 状态字节，错误码 0x0002 会被报成两度或者两转。
func (b *Backend) sysInfo(query uint32, wide bool) (int, error) {
	outputs, err := b.callGaming("GetGamingSysInfo", map[string]wmi.Value{
		"gmInput": wmi.Uint32(query),
	})
	if err != nil {
		return 0, err
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("hardware: GetGamingSysInfo 没有返回 gmOutput")
	}

	raw, err := decodeUint(value)
	if err != nil {
		return 0, err
	}
	if status := raw & 0xFF; status != 0 {
		return 0, fmt.Errorf("hardware: 固件没有回答分类 0x%02X，状态字节是 %d", query>>8, status)
	}
	if wide {
		return int((raw >> 8) & 0xFFFF), nil
	}
	return int((raw >> 8) & 0xFF), nil
}

// decodeUint extracts a numeric value from a WMI result.
//
// The Acer provider returns most readings as a hexadecimal string, so both
// representations have to be accepted.
func decodeUint(value wmi.Value) (uint64, error) {
	if value.Kind == wmi.KindString {
		text := strings.TrimSpace(value.Str)
		if text == "" {
			return 0, nil
		}
		base := 10
		if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
			text = text[2:]
			base = 16
		}
		n, err := strconv.ParseUint(text, base, 64)
		if err != nil {
			return 0, fmt.Errorf("hardware: cannot parse %q: %w", value.Str, err)
		}
		return n, nil
	}
	return value.UInt64(), nil
}

// decodeBytes extracts a byte array from a WMI result.
//
// The provider answers the methods that carry a buffer with an array of
// bytes, some builds answer with the same bytes as a hexadecimal text.
func decodeBytes(value wmi.Value) ([]byte, error) {
	switch value.Kind {
	case wmi.KindBytes:
		return value.Bytes, nil
	case wmi.KindString:
		text := strings.TrimSpace(value.Str)
		text = strings.NewReplacer(" ", "", "-", "", ":", "", "\t", "").Replace(text)
		if text == "" {
			return nil, nil
		}
		if len(text)%2 != 0 {
			return nil, fmt.Errorf("hardware: %q 的位数不是偶数，不是字节数组", value.Str)
		}
		out := make([]byte, 0, len(text)/2)
		for i := 0; i < len(text); i += 2 {
			n, err := strconv.ParseUint(text[i:i+2], 16, 8)
			if err != nil {
				return nil, fmt.Errorf("hardware: cannot parse %q: %w", value.Str, err)
			}
			out = append(out, byte(n))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("hardware: 答案不是字节数组")
	}
}
