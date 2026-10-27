package agentservice

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// request is a request that the fake service received.
type request struct {
	// Packet is the identifier of the packet.
	Packet uint32
	// Function is the name of the setting.
	Function string
	// Parameter is the value that was written.
	Parameter any
	// Raw is the payload of the request as it came in.
	Raw map[string]any
}

// fake is a service that answers the requests of a test.
//
// The answers are given per function, a function that has no answer is
// answered with the failure of the service.
type fake struct {
	listener net.Listener

	mutex sync.Mutex
	// answers maps a function to the answer it gets. The key of the
	// monitor data and of the devices is the name of the packet.
	answers map[string]map[string]any
	// received holds the requests that arrived, in order.
	received []request
	// breakAfter closes the connection after this many requests. A value
	// of zero keeps the connection open.
	breakAfter int
	// requests counts the requests that arrived.
	requests int
	// conns holds the connections that are open.
	conns []net.Conn
}

// stop closes the service and the connections it holds.
func (f *fake) stop() {
	f.mutex.Lock()
	conns := make([]net.Conn, len(f.conns))
	copy(conns, f.conns)
	f.mutex.Unlock()

	f.listener.Close()
	for _, conn := range conns {
		conn.Close()
	}
}

func newFake(t *testing.T, answers map[string]map[string]any) *fake {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	f := &fake{listener: listener, answers: answers}
	go f.serve()
	t.Cleanup(func() { listener.Close() })
	return f
}

// address is the address of the fake service.
func (f *fake) address() string { return f.listener.Addr().String() }

// serve answers the requests that arrive.
func (f *fake) serve() {
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			return
		}
		go f.handle(conn)
	}
}

// handle answers the requests of one connection.
func (f *fake) handle(conn net.Conn) {
	defer conn.Close()

	f.mutex.Lock()
	f.conns = append(f.conns, conn)
	f.mutex.Unlock()

	for {
		received, err := readFrame(conn)
		if err != nil {
			return
		}

		f.mutex.Lock()
		f.received = append(f.received, received)
		f.requests++
		broken := f.breakAfter > 0 && f.requests > f.breakAfter
		f.mutex.Unlock()

		if err := writeAnswer(conn, f.answerFor(received)); err != nil {
			return
		}
		if broken {
			return
		}
	}
}

// answerFor builds the answer of a request.
func (f *fake) answerFor(received request) map[string]any {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	key := received.Function
	if key == "" {
		key = fmt.Sprintf("packet:%d", received.Packet)
	}
	answer, ok := f.answers[key]
	if !ok {
		return map[string]any{"request": received.Function, "result": ResultFailed}
	}
	return answer
}

// requests returns the requests that arrived.
func (f *fake) requests_() []request {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	out := make([]request, len(f.received))
	copy(out, f.received)
	return out
}

// readFrame reads one request of a connection.
func readFrame(conn net.Conn) (request, error) {
	header := make([]byte, 8)
	if _, err := readFull(conn, header); err != nil {
		return request{}, err
	}
	if string(header[:4]) != magic {
		return request{}, fmt.Errorf("请求没有以 %s 开头", magic)
	}

	body, err := readJSON(conn)
	if err != nil {
		return request{}, err
	}

	var payload map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			return request{}, err
		}
	}

	out := request{
		Packet:   binary.LittleEndian.Uint32(header[4:]),
		Function: stringOf(payload["Function"]),
		Raw:      payload,
	}
	if parameter, ok := payload["Parameter"]; ok {
		out.Parameter = parameter
	}
	return out, nil
}

// readFull reads a fixed number of bytes.
func readFull(conn net.Conn, buffer []byte) (int, error) {
	read := 0
	for read < len(buffer) {
		n, err := conn.Read(buffer[read:])
		if err != nil {
			return read, err
		}
		read += n
	}
	return read, nil
}

// readJSON reads one JSON document of a stream.
func readJSON(conn net.Conn) ([]byte, error) {
	var out []byte
	depth := 0
	started := false
	buffer := make([]byte, 1)

	for {
		_, err := conn.Read(buffer)
		if err != nil {
			return out, err
		}
		out = append(out, buffer[0])

		switch buffer[0] {
		case '{':
			depth++
			started = true
		case '}':
			depth--
			if started && depth == 0 {
				return out, nil
			}
		}
	}
}

// writeAnswer writes the answer of a request.
func writeAnswer(conn net.Conn, answer map[string]any) error {
	body, err := json.Marshal(answer)
	if err != nil {
		return err
	}
	_, err = conn.Write(body)
	return err
}

// stringOf reads a string of a value.
func stringOf(value any) string {
	text, _ := value.(string)
	return text
}

func TestGetReadsTheSettingOfTheService(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		FunctionLighting: {
			"data":    map[string]any{"brightness": 100},
			"request": FunctionLighting,
			"result":  ResultOK,
		},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	answer, err := client.Get(FunctionLighting)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if err, failed := answer.Err(); failed {
		t.Fatalf("读取不该失败: %v", err)
	}

	var data struct {
		Brightness int `json:"brightness"`
	}
	if err := answer.Decode(&data); err != nil {
		t.Fatalf("数据无法解析: %v", err)
	}
	if data.Brightness != 100 {
		t.Fatalf("亮度是 %d，不是 100", data.Brightness)
	}

	requests := service.requests_()
	if len(requests) != 1 {
		t.Fatalf("服务收到了 %d 个请求，不是 1 个", len(requests))
	}
	if requests[0].Packet != PacketGetUpdatedData {
		t.Fatalf("包标识是 %d，不是 %d", requests[0].Packet, PacketGetUpdatedData)
	}
	if requests[0].Function != FunctionLighting {
		t.Fatalf("请求的设置名是 %q，不是 %q", requests[0].Function, FunctionLighting)
	}
}

func TestSetSendsTheParameter(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		FunctionOperatingMode: {"request": FunctionOperatingMode, "result": ResultOK},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	answer, err := client.Set(FunctionOperatingMode, map[string]any{"mode": 4})
	if err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	if err, failed := answer.Err(); failed {
		t.Fatalf("写入不该失败: %v", err)
	}

	requests := service.requests_()
	if len(requests) != 1 {
		t.Fatalf("服务收到了 %d 个请求，不是 1 个", len(requests))
	}
	if requests[0].Packet != PacketSetDeviceData {
		t.Fatalf("包标识是 %d，不是 %d", requests[0].Packet, PacketSetDeviceData)
	}

	parameter, ok := requests[0].Parameter.(map[string]any)
	if !ok {
		t.Fatalf("参数不是一组数值: %#v", requests[0].Parameter)
	}
	if mode, ok := parameter["mode"].(float64); !ok || mode != 4 {
		t.Fatalf("参数里的档位是 %#v，不是 4", parameter["mode"])
	}
}

func TestUnsupportedRequestIsReported(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		CapabilityOverclockingMode: {
			"request": CapabilityOverclockingMode,
			"result":  ResultUnsupported,
		},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	answer, err := client.Capability(CapabilityOverclockingMode)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	err, failed := answer.Err()
	if !failed {
		t.Fatal("不支持的请求不该算作成功")
	}
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrUnsupported)
	}
	if len(answer.Data) != 0 {
		t.Fatalf("不支持的应答不该带数据: %s", answer.Data)
	}
}

func TestFailedWriteIsReported(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		FunctionStickyKey: {"request": FunctionStickyKey, "result": ResultFailed},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	answer, err := client.Set(FunctionStickyKey, map[string]any{"status": 1})
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	err, failed := answer.Err()
	if !failed {
		t.Fatal("失败的写入不该算作成功")
	}
	if !errors.Is(err, ErrFailed) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrFailed)
	}
}

func TestOneConnectionCarriesManyRequests(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		FunctionOperatingMode: {"data": map[string]any{"mode": 1}, "request": FunctionOperatingMode, "result": ResultOK},
		FunctionGPUMode:       {"data": map[string]any{"mode": 1}, "request": FunctionGPUMode, "result": ResultOK},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	for i := 0; i < 3; i++ {
		if _, err := client.Get(FunctionOperatingMode); err != nil {
			t.Fatalf("第 %d 次读取失败: %v", i+1, err)
		}
		if _, err := client.Get(FunctionGPUMode); err != nil {
			t.Fatalf("第 %d 次读取失败: %v", i+1, err)
		}
	}

	if count := len(service.requests_()); count != 6 {
		t.Fatalf("服务收到了 %d 个请求，不是 6 个", count)
	}
}

func TestBrokenConnectionIsOpenedAgain(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		FunctionOperatingMode: {"data": map[string]any{"mode": 1}, "request": FunctionOperatingMode, "result": ResultOK},
	})
	service.breakAfter = 1

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	if _, err := client.Get(FunctionOperatingMode); err != nil {
		t.Fatalf("第一次读取失败: %v", err)
	}
	// The service closes the connection after the first request, the client
	// has to open a new one for the second.
	if _, err := client.Get(FunctionOperatingMode); err != nil {
		t.Fatalf("第二次读取失败: %v", err)
	}
}

func TestAnswerWithBracesInAString(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		FunctionLighting: {
			"data":    map[string]any{"color": "#{}FF00"},
			"request": FunctionLighting,
			"result":  ResultOK,
		},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	answer, err := client.Get(FunctionLighting)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}

	var data struct {
		Color string `json:"color"`
	}
	if err := answer.Decode(&data); err != nil {
		t.Fatalf("数据无法解析: %v", err)
	}
	if data.Color != "#{}FF00" {
		t.Fatalf("颜色是 %q，不是 %q", data.Color, "#{}FF00")
	}
}

func TestServiceThatIsNotThere(t *testing.T) {
	client, err := Connect(Options{Address: "127.0.0.1:1", Timeout: time.Second})
	if err == nil {
		client.Close()
		t.Fatal("连接不上的服务不该算作连接成功")
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrUnavailable)
	}
}

func TestMonitorDataAsksForThePacketOfTheReadings(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		"packet:10": {
			"data":    map[string]any{"CPU_TEMPERATURE": 91},
			"request": "GET_MONITOR_DATA",
			"result":  ResultOK,
		},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	answer, err := client.MonitorData()
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}

	var data struct {
		CPUTemperature int `json:"CPU_TEMPERATURE"`
	}
	if err := answer.Decode(&data); err != nil {
		t.Fatalf("数据无法解析: %v", err)
	}
	if data.CPUTemperature != 91 {
		t.Fatalf("CPU 温度是 %d，不是 91", data.CPUTemperature)
	}

	requests := service.requests_()
	if len(requests) != 1 {
		t.Fatalf("服务收到了 %d 个请求，不是 1 个", len(requests))
	}
	if requests[0].Packet != PacketMonitorData {
		t.Fatalf("包标识是 %d，不是 %d", requests[0].Packet, PacketMonitorData)
	}
	if requests[0].Function != "" {
		t.Fatalf("监视数据的请求不该带设置名: %q", requests[0].Function)
	}
}

func TestDecodeWithoutData(t *testing.T) {
	answer := Answer{Request: FunctionStickyKey, Result: ResultFailed}

	var data map[string]any
	if err := answer.Decode(&data); err == nil {
		t.Fatal("没有数据的应答不该解析成功")
	} else if !strings.Contains(err.Error(), FunctionStickyKey) {
		t.Fatalf("错误里没有请求的名字: %v", err)
	}
}

func TestCloseReleasesTheConnection(t *testing.T) {
	service := newFake(t, map[string]map[string]any{})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("关闭失败: %v", err)
	}
	// Closing twice is not an error.
	if err := client.Close(); err != nil {
		t.Fatalf("第二次关闭失败: %v", err)
	}
}

func TestDevicesAsksForThePacketOfTheDevices(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		"packet:30": {
			"data":    []any{map[string]any{"device_index": 0}},
			"request": "GET_CONNECTED_DEVICE",
			"result":  ResultOK,
		},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	if _, err := client.Devices(); err != nil {
		t.Fatalf("读取设备失败: %v", err)
	}

	requests := service.requests_()
	if len(requests) != 1 {
		t.Fatalf("服务收到了 %d 个请求，不是 1 个", len(requests))
	}
	if requests[0].Packet != PacketGetConnectedDevice {
		t.Fatalf("包标识是 %d，不是 %d", requests[0].Packet, PacketGetConnectedDevice)
	}
}

func TestCapabilityAsksForThePacketOfTheAbilities(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		CapabilityVersion: {
			"data":    "1.2.2101.0",
			"request": CapabilityVersion,
			"result":  ResultOK,
		},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	answer, err := client.Capability(CapabilityVersion)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if string(answer.Data) != `"1.2.2101.0"` {
		t.Fatalf("版本是 %s，不是 1.2.2101.0", answer.Data)
	}

	requests := service.requests_()
	if requests[0].Packet != PacketInitialization {
		t.Fatalf("包标识是 %d，不是 %d", requests[0].Packet, PacketInitialization)
	}
}

func TestAnAnswerWithAnUnknownResult(t *testing.T) {
	answer := Answer{Request: FunctionLighting, Result: "9"}

	err, failed := answer.Err()
	if !failed {
		t.Fatal("不认识的编号不该算作成功")
	}
	if !errors.Is(err, ErrFailed) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrFailed)
	}
}

func TestAnAnswerWithoutAResult(t *testing.T) {
	answer := Answer{Request: FunctionLighting}

	if !answer.OK() {
		t.Fatal("没有编号的应答不该算作失败")
	}
	err, failed := answer.Err()
	if failed || err != nil {
		t.Fatalf("没有编号的应答不该带错误: %v", err)
	}
}

func TestDataThatCannotBeParsed(t *testing.T) {
	answer := Answer{
		Data:   json.RawMessage(`{"mode": "很多"}`),
		Result: ResultOK,
	}

	var data struct {
		Mode int `json:"mode"`
	}
	if err := answer.Decode(&data); err == nil {
		t.Fatal("解析不了的据不该算作成功")
	}
}

func TestAParameterThatCannotBeSerialized(t *testing.T) {
	_, err := buildFrame(PacketSetDeviceData, FunctionLighting, make(chan int))
	if err == nil {
		t.Fatal("序列化不了的参数不该算作成功")
	}
}

func TestAnAnswerWithAnEscapedQuote(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		FunctionLighting: {
			"data":    map[string]any{"note": `他说"你好"`},
			"request": FunctionLighting,
			"result":  ResultOK,
		},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	answer, err := client.Get(FunctionLighting)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}

	var data struct {
		Note string `json:"note"`
	}
	if err := answer.Decode(&data); err != nil {
		t.Fatalf("数据无法解析: %v", err)
	}
	if data.Note != `他说"你好"` {
		t.Fatalf("读回来的是 %q", data.Note)
	}
}

func TestAServiceThatGoesAway(t *testing.T) {
	service := newFake(t, map[string]map[string]any{
		FunctionOperatingMode: {
			"data":    map[string]any{"mode": 1},
			"request": FunctionOperatingMode,
			"result":  ResultOK,
		},
	})

	client, err := Connect(Options{Address: service.address()})
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	if _, err := client.Get(FunctionOperatingMode); err != nil {
		t.Fatalf("第一次读取失败: %v", err)
	}

	// The service goes away, the client has to report it instead of
	// pretending that the answer is fine.
	service.stop()

	if _, err := client.Get(FunctionOperatingMode); err == nil {
		t.Fatal("服务没了之后该报错")
	}
}
