// Package agentservice talks to the original AcerAgentService.
//
// The service is the heart of the original PredatorSense stack: it owns the
// keyboard lighting, the fans, the performance profiles and the readings of
// the machine, and it hands the lighting on to AcerLightingService.
//
// The service listens on a local port and answers with a JSON document. A
// request is the word ACER, the identifier of the packet as a little endian
// word and the payload as a JSON document:
//
//	{"Function": "LIGHTING", "Parameter": ...}
//
// An answer carries the name of the request, a result code and, when the
// service knows the request, the data of it:
//
//	{"data": ..., "request": "LIGHTING", "result": "0"}
//
// The result code is the way the service says whether it did the work:
// "0" is done, "3" is a request it does not implement and "4" is a request
// it did not carry out.
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

// The constants of the protocol.
const (
	// defaultPort is the port of the AcerAgentService.
	defaultPort = 46933

	// magic is the word every request starts with.
	magic = "ACER"

	// defaultTimeout is the time that a single request may take.
	defaultTimeout = 5 * time.Second
)

// The packet identifiers of the protocol.
const (
	// PacketInitialization asks for the abilities of the machine.
	PacketInitialization = 0
	// PacketMonitorData asks for the live numbers of the machine.
	PacketMonitorData = 10
	// PacketGetUpdatedData asks for the value of a setting.
	PacketGetUpdatedData = 20
	// PacketGetConnectedDevice asks for the devices that carry a lighting.
	PacketGetConnectedDevice = 30
	// PacketGetLightingEffect asks for the abilities of the lighting.
	PacketGetLightingEffect = 40
	// PacketSetDeviceData writes a setting.
	PacketSetDeviceData = 100
)

// The result codes of an answer.
const (
	// ResultOK is the code of an answer that carries the result of the work.
	ResultOK = "0"
	// ResultUnsupported is the code of a request the service does not implement.
	ResultUnsupported = "3"
	// ResultFailed is the code of a request the service did not carry out.
	ResultFailed = "4"
)

// The names of the settings the service knows.
const (
	// FunctionOperatingMode is the performance profile.
	FunctionOperatingMode = "OPERATING_MODE"
	// FunctionGPUMode is the graphics switch.
	FunctionGPUMode = "GPU_MODE"
	// FunctionLighting is the keyboard lighting.
	FunctionLighting = "LIGHTING"
	// FunctionFanControl is the fan configuration.
	FunctionFanControl = "FAN_CONTROL"
	// FunctionWinKey is the Windows key.
	FunctionWinKey = "WIN_KEY"
	// FunctionLCDOverdrive is the overdrive of the display.
	FunctionLCDOverdrive = "LCD_OVERDRIVE"
	// FunctionBootSound is the sound at the start of the machine.
	FunctionBootSound = "BOOT_SOUND"
	// FunctionBacklightTimeout is the automatic switch off of the backlight.
	FunctionBacklightTimeout = "BACKLIGHT_30S"
	// FunctionStickyKey is the sticky keys.
	FunctionStickyKey = "STICKY_KEY"
	// FunctionBatteryBoost is the battery boost.
	FunctionBatteryBoost = "BATTERY_BOOST"
	// FunctionSoundMode is the sound profile.
	FunctionSoundMode = "SOUND_MODE"
	// FunctionCustomBootLogo is the boot logo.
	FunctionCustomBootLogo = "CUSTOM_BOOT_LOGO"
	// FunctionAdaptorStatus is the state of the power supply.
	FunctionAdaptorStatus = "ADAPTOR_STATUS"
)

// The names of the abilities the service reports.
const (
	// CapabilityVersion is the version of the service.
	CapabilityVersion = "VERSION"
	// CapabilityOperatingMode is the list of the performance profiles.
	CapabilityOperatingMode = "SUPPORT_OPERATING_MODE_CAPABILITY"
	// CapabilityAudioMode is the list of the sound profiles.
	CapabilityAudioMode = "SUPPORT_AUDIO_MODE"
	// CapabilityGPUMode is the list of the graphics modes.
	CapabilityGPUMode = "SUPPORT_GPU_MODE"
	// CapabilityGPUModeCapability is the list of the graphics switch abilities.
	CapabilityGPUModeCapability = "SUPPORT_GPU_MODE_CAPABILITY"
	// CapabilityLCDOverdrive is the ability of the display overdrive.
	CapabilityLCDOverdrive = "SUPPORT_LCD_OVERDRIVE"
	// CapabilityCustomBootLogo is the ability of the boot logo.
	CapabilityCustomBootLogo = "SUPPORT_CUSTOM_BOOT_LOGO"
	// CapabilityKeyboardInfo describes the keyboard.
	CapabilityKeyboardInfo = "SUPPORT_KEYBOARD_INFO"
	// CapabilityFanControl describes the fans.
	CapabilityFanControl = "SUPPORT_FAN_CONTROL"
	// CapabilityCPUInfo describes the processor.
	CapabilityCPUInfo = "SUPPORT_CPU_INFO"
	// CapabilityGPUInfo describes the graphics card.
	CapabilityGPUInfo = "SUPPORT_GPU_INFO"
	// CapabilityOverclockingMode is the ability of the overclocking.
	CapabilityOverclockingMode = "SUPPORT_OVERCLOCKING_MODE_CAPABILITY"
)

// The errors of the client.
var (
	// ErrUnsupported says that the service does not implement the request.
	ErrUnsupported = errors.New("agentservice: 原厂服务不支持这个请求")

	// ErrFailed says that the service did not carry out the request.
	ErrFailed = errors.New("agentservice: 原厂服务没有执行这次操作")

	// ErrUnavailable says that the service cannot be reached.
	ErrUnavailable = errors.New("agentservice: 原厂服务不可达")
)

// Answer is the answer of the service.
//
// The data is empty when the service does not carry out the request, such
// answers carry the result code only.
type Answer struct {
	// Data is the payload of the answer. It is empty when the answer has
	// no data field at all.
	Data json.RawMessage `json:"data"`
	// Request is the name of the request the answer belongs to.
	Request string `json:"request"`
	// Result is the result code of the work.
	Result string `json:"result"`
}

// OK tells whether the service did the work.
func (a Answer) OK() bool {
	return a.Result == "" || a.Result == ResultOK
}

// Err turns the result code of the answer into an error.
//
// The second result is false when the answer is a success, so that a caller
// can tell the difference between a request the service refuses and one it
// does not know.
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

// Decode reads the data of the answer into a value.
func (a Answer) Decode(target any) error {
	if len(a.Data) == 0 {
		return fmt.Errorf("agentservice: 应答 %s 没有数据", a.Request)
	}
	if err := json.Unmarshal(a.Data, target); err != nil {
		return fmt.Errorf("agentservice: 应答 %s 的数据无法解析: %w", a.Request, err)
	}
	return nil
}

// Options change the behaviour of a client.
type Options struct {
	// Address is the address of the service. The default is the local port
	// of the AcerAgentService.
	Address string
	// Timeout limits the time of a single request.
	Timeout time.Duration
}

// Client is a connection to the AcerAgentService.
//
// One connection carries many requests: the service reads a request, writes
// an answer and waits for the next one. The client keeps the connection open
// and opens a new one when the old one breaks.
type Client struct {
	address string
	timeout time.Duration

	// mutex guards the connection, because the requests of the program run
	// side by side and a connection carries one request at a time.
	mutex sync.Mutex
	conn  net.Conn
	read  *bufio.Reader
}

// Connect opens a connection to the AcerAgentService.
//
// The service has to be running, a machine without it is answered with an
// error that says so.
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

// open dials the service.
//
// The caller holds the lock.
func (c *Client) open() error {
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrUnavailable, c.address, err)
	}
	c.conn = conn
	c.read = bufio.NewReader(conn)
	return nil
}

// Close releases the connection.
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

// Capability asks for an ability of the machine.
func (c *Client) Capability(function string) (Answer, error) {
	return c.ask(PacketInitialization, function, nil)
}

// MonitorData asks for the live numbers of the machine.
func (c *Client) MonitorData() (Answer, error) {
	return c.ask(PacketMonitorData, "", nil)
}

// Get asks for the value of a setting.
func (c *Client) Get(function string) (Answer, error) {
	return c.ask(PacketGetUpdatedData, function, nil)
}

// Devices asks for the devices that carry a lighting.
func (c *Client) Devices() (Answer, error) {
	return c.ask(PacketGetConnectedDevice, "", nil)
}

// Set writes a setting.
//
// The parameter is the value the official application sends, the shape of it
// depends on the setting.
func (c *Client) Set(function string, parameter any) (Answer, error) {
	return c.ask(PacketSetDeviceData, function, parameter)
}

// ask sends one request and reads its answer.
//
// The connection is opened again when the old one broke, so that a service
// that was restarted does not take the program down with it.
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

	// The connection may have been broken by a service that was restarted.
	// A second try on a fresh connection tells the two apart.
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

// exchange writes a frame and reads the answer of it.
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

// readAnswer reads one JSON document of the stream.
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

// buildFrame builds the frame of a request.
//
// The frame is the word, the identifier of the packet as a little endian
// word and the payload as a JSON document.
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
