// Package agentservice 提供与 AcerAgentService 后台服务的底层 TCP 通信客户端。
package agentservice

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	defaultPort    = 46933
	magic          = "ACER"
	defaultTimeout = 5 * time.Second
)

// 数据包操作类型代码。
const (
	// PacketInitialization 用于初始化与能力查询。
	PacketInitialization = 0
	// PacketMonitorData 用于获取实时监控遥测数据。
	PacketMonitorData = 10
	// PacketGetUpdatedData 用于查询特定功能当前状态。
	PacketGetUpdatedData = 20
	// PacketGetConnectedDevice 用于枚举连接的受支持设备。
	PacketGetConnectedDevice = 30
	// PacketGetLightingEffect 用于获取灯效配置。
	PacketGetLightingEffect = 40
	// PacketSetDeviceData 用于设置设备功能参数。
	PacketSetDeviceData = 100
)

// 服务端响应结果标识。
const (
	ResultOK          = "0"
	ResultUnsupported = "3"
	ResultFailed      = "4"
)

// 原厂服务支持的功能标识常量。
const (
	FunctionOperatingMode    = "OPERATING_MODE"
	FunctionGPUMode          = "GPU_MODE"
	FunctionLighting         = "LIGHTING"
	FunctionFanControl       = "FAN_CONTROL"
	FunctionWinKey           = "WIN_KEY"
	FunctionLCDOverdrive     = "LCD_OVERDRIVE"
	FunctionBootSound        = "BOOT_SOUND"
	FunctionBacklightTimeout = "BACKLIGHT_30S"
	FunctionStickyKey        = "STICKY_KEY"
	FunctionBatteryBoost     = "BATTERY_BOOST"
	FunctionSoundMode        = "SOUND_MODE"
	FunctionCustomBootLogo   = "CUSTOM_BOOT_LOGO"
	FunctionAdaptorStatus    = "ADAPTOR_STATUS"
)

// 原厂服务能力查询标识常量。
const (
	CapabilityVersion           = "VERSION"
	CapabilityOperatingMode     = "SUPPORT_OPERATING_MODE_CAPABILITY"
	CapabilityAudioMode         = "SUPPORT_AUDIO_MODE"
	CapabilityGPUMode           = "SUPPORT_GPU_MODE"
	CapabilityGPUModeCapability = "SUPPORT_GPU_MODE_CAPABILITY"
	CapabilityLCDOverdrive      = "SUPPORT_LCD_OVERDRIVE"
	CapabilityCustomBootLogo    = "SUPPORT_CUSTOM_BOOT_LOGO"
	CapabilityKeyboardInfo      = "SUPPORT_KEYBOARD_INFO"
	CapabilityFanControl        = "SUPPORT_FAN_CONTROL"
	CapabilityCPUInfo           = "SUPPORT_CPU_INFO"
	CapabilityGPUInfo           = "SUPPORT_GPU_INFO"
	CapabilityOverclockingMode  = "SUPPORT_OVERCLOCKING_MODE_CAPABILITY"
)

var (
	// ErrUnsupported 表示目标功能不被当前机器或服务支持。
	ErrUnsupported = errors.New("agentservice: 原厂服务不支持这个请求")
	// ErrFailed 表示原厂服务拒绝或未能成功执行请求。
	ErrFailed = errors.New("agentservice: 原厂服务没有执行这次操作")
	// ErrUnavailable 表示无法连接至原厂服务进程。
	ErrUnavailable = errors.New("agentservice: 原厂服务不可达")
)

// Answer 表示从 AcerAgentService 接收到的结构化响应。
type Answer struct {
	Data    json.RawMessage `json:"data"`
	Request string          `json:"request"`
	Result  string          `json:"result"`
}

// OK 判断响应是否成功。
func (a Answer) OK() bool {
	return a.Result == "" || a.Result == ResultOK
}

// Err 将服务返回的非零结果转换为 Go 错误对象。
func (a Answer) Err() (error, bool) {
	switch a.Result {
	case "", ResultOK:
		return nil, false
	case ResultUnsupported:
		return fmt.Errorf("%w: %s", ErrUnsupported, a.Request), true
	default:
		return fmt.Errorf("%w: %s（结果 %s）", ErrFailed, a.Request, a.Result), true
	}
}

// Decode 将响应数据中的 JSON 反序列化到目标变量中。
func (a Answer) Decode(target any) error {
	if len(a.Data) == 0 {
		return fmt.Errorf("agentservice: 应答 %s 没有数据", a.Request)
	}
	if err := json.Unmarshal(a.Data, target); err != nil {
		return fmt.Errorf("agentservice: 应答 %s 的数据无法解析: %w", a.Request, err)
	}
	return nil
}

// Options 定义连接 AcerAgentService 的配置参数。
type Options struct {
	Address string
	Timeout time.Duration
}

// Client 封装与 AcerAgentService 的长连接与请求交互。
type Client struct {
	address string
	timeout time.Duration

	mutex sync.Mutex
	conn  net.Conn
	read  *bufio.Reader
}

// Connect 建立与 AcerAgentService 的 TCP 连接。
func Connect(options Options) (*Client, error) {
	address := options.Address
	if address == "" {
		address = fmt.Sprintf("127.0.0.1:%d", defaultPort)
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	client := &Client{address: address, timeout: timeout}
	if err := client.open(); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) open() error {
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrUnavailable, c.address, err)
	}
	c.conn = conn
	c.read = bufio.NewReader(conn)
	return nil
}

// Close 关闭与服务端的网络连接。
func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.read = nil
	return err
}

// Capability 查询指定功能的支持情况。
func (c *Client) Capability(function string) (Answer, error) {
	return c.ask(PacketInitialization, function, nil)
}

// MonitorData 查询系统监控与传感器实时数据包。
func (c *Client) MonitorData() (Answer, error) {
	return c.ask(PacketMonitorData, "", nil)
}

// Get 查询指定功能的当前设置值。
func (c *Client) Get(function string) (Answer, error) {
	return c.ask(PacketGetUpdatedData, function, nil)
}

// GetFrom 查询指定设备上的当前设置值。
//
// 真机实测：灯光的读取需要带上设备号，否则服务只回 0 号设备的设置，
// 读不到分区直控那一份。
func (c *Client) GetFrom(function string, device int) (Answer, error) {
	return c.ask(PacketGetUpdatedData, function, map[string]any{"device": device})
}

// Devices 枚举当前接入的设备列表。
func (c *Client) Devices() (Answer, error) {
	return c.ask(PacketGetConnectedDevice, "", nil)
}

// Set 向指定功能下发新的控制参数。
func (c *Client) Set(function string, parameter any) (Answer, error) {
	return c.ask(PacketSetDeviceData, function, parameter)
}

func (c *Client) ask(packet uint32, function string, parameter any) (Answer, error) {
	frame, err := buildFrame(packet, function, parameter)
	if err != nil {
		return Answer{}, err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	answer, err := c.exchange(frame)
	if err == nil {
		return answer, nil
	}

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		c.read = nil
	}
	if openErr := c.open(); openErr != nil {
		return Answer{}, err
	}
	return c.exchange(frame)
}

func (c *Client) exchange(frame []byte) (Answer, error) {
	if c.conn == nil {
		if err := c.open(); err != nil {
			return Answer{}, err
		}
	}

	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	if _, err := c.conn.Write(frame); err != nil {
		return Answer{}, fmt.Errorf("agentservice: 请求写入失败: %w", err)
	}

	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	raw, err := c.readAnswer()
	if err != nil {
		return Answer{}, fmt.Errorf("agentservice: 应答读取失败: %w", err)
	}

	var answer Answer
	if err := json.Unmarshal(raw, &answer); err != nil {
		return Answer{}, fmt.Errorf("agentservice: 应答无法解析: %w", err)
	}
	return answer, nil
}

func (c *Client) readAnswer() ([]byte, error) {
	var out bytes.Buffer
	depth := 0
	started := false
	inString := false
	escaped := false

	for {
		b, err := c.read.ReadByte()
		if err != nil {
			return nil, err
		}
		out.WriteByte(b)

		if inString {
			switch {
			case escaped:
				escaped = false
			case b == '\\':
				escaped = true
			case b == '"':
				inString = false
			}
			continue
		}

		switch b {
		case '"':
			inString = true
		case '{':
			depth++
			started = true
		case '}':
			depth--
			if started && depth == 0 {
				return out.Bytes(), nil
			}
		}
	}
}

func buildFrame(packet uint32, function string, parameter any) ([]byte, error) {
	payload := map[string]any{}
	if function != "" {
		payload["Function"] = function
	}
	if parameter != nil {
		payload["Parameter"] = parameter
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("agentservice: 请求内容无法序列化: %w", err)
	}

	frame := make([]byte, 0, len(body)+8)
	frame = append(frame, magic...)
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], packet)
	frame = append(frame, header[:]...)
	frame = append(frame, body...)
	return frame, nil
}
