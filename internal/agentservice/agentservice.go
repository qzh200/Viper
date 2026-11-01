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

const (
	PacketInitialization     = 0
	PacketMonitorData        = 10
	PacketGetUpdatedData     = 20
	PacketGetConnectedDevice = 30
	PacketGetLightingEffect  = 40
	PacketSetDeviceData      = 100
)

const (
	ResultOK          = "0"
	ResultUnsupported = "3"
	ResultFailed      = "4"
)

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
	ErrUnsupported = errors.New("agentservice: 原厂服务不支持这个请求")
	ErrFailed      = errors.New("agentservice: 原厂服务没有执行这次操作")
	ErrUnavailable = errors.New("agentservice: 原厂服务不可达")
)

type Answer struct {
	Data    json.RawMessage `json:"data"`
	Request string          `json:"request"`
	Result  string          `json:"result"`
}

func (a Answer) OK() bool {
	return a.Result == "" || a.Result == ResultOK
}

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

func (a Answer) Decode(target any) error {
	if len(a.Data) == 0 {
		return fmt.Errorf("agentservice: 应答 %s 没有数据", a.Request)
	}
	if err := json.Unmarshal(a.Data, target); err != nil {
		return fmt.Errorf("agentservice: 应答 %s 的数据无法解析: %w", a.Request, err)
	}
	return nil
}

type Options struct {
	Address string
	Timeout time.Duration
}

type Client struct {
	address string
	timeout time.Duration

	mutex sync.Mutex
	conn  net.Conn
	read  *bufio.Reader
}

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

func (c *Client) Capability(function string) (Answer, error) {
	return c.ask(PacketInitialization, function, nil)
}

func (c *Client) MonitorData() (Answer, error) {
	return c.ask(PacketMonitorData, "", nil)
}

func (c *Client) Get(function string) (Answer, error) {
	return c.ask(PacketGetUpdatedData, function, nil)
}

func (c *Client) GetFrom(function string, device int) (Answer, error) {
	return c.ask(PacketGetUpdatedData, function, map[string]any{"device": device})
}

func (c *Client) Devices() (Answer, error) {
	return c.ask(PacketGetConnectedDevice, "", nil)
}

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
