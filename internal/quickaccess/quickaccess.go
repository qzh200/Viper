// Package quickaccess 实现与 Acer QuickAccess 服务的 TLS/WebSocket 安全握手与命令交互。
package quickaccess

import (
	"bufio"
	"crypto/aes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	defaultPort = 5141

	packetAuth     = 1
	packetRequest  = 2
	packetResponse = 3

	keyA = "A6052DC8A6E44210"
	keyB = "AB252AB73BED1CDB"

	question = "AcerQuickAccessFunctionalityTests_HandshakingQuestion"

	defaultTimeout = 5 * time.Second
)

// 交互操作动作标识。
const (
	// ActionGet 表示查询状态操作。
	ActionGet = "Get"
	// ActionSet 表示写入设置操作。
	ActionSet = "Set"
)

// Client 维护与 Acer QuickAccess 服务的加密连接。
type Client struct {
	conn net.Conn
	read *bufio.Reader
	host string

	timeout time.Duration
}

// Options 定义连接 QuickAccess 服务的可选配置。
type Options struct {
	Address string
	Timeout time.Duration
}

// Connect 建立与 QuickAccess 服务的 TLS 连接，完成 WebSocket 升级并执行 AES 鉴权握手。
func Connect(options Options) (*Client, error) {
	address := options.Address
	if address == "" {
		address = fmt.Sprintf("127.0.0.1:%d", defaultPort)
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: timeout},
		"tcp",
		address,
		&tls.Config{InsecureSkipVerify: true},
	)
	if err != nil {
		return nil, fmt.Errorf("quickaccess: connect to %s: %w", address, err)
	}

	client := &Client{
		conn:    conn,
		read:    bufio.NewReader(conn),
		host:    address,
		timeout: timeout,
	}
	if err := client.upgrade(); err != nil {
		conn.Close()
		return nil, err
	}
	if err := client.handshake(); err != nil {
		conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *Client) upgrade() error {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return err
	}

	request := "GET / HTTP/1.1\r\n" +
		"Host: " + c.host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + base64.StdEncoding.EncodeToString(key) + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"

	if c.timeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	}
	if _, err := io.WriteString(c.conn, request); err != nil {
		return fmt.Errorf("quickaccess: write websocket upgrade: %w", err)
	}

	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	status, err := c.read.ReadString('\n')
	if err != nil {
		return fmt.Errorf("quickaccess: read websocket upgrade status: %w", err)
	}
	if !strings.Contains(status, "101") {
		return fmt.Errorf("quickaccess: upgrade rejected: %s", strings.TrimSpace(status))
	}

	for {
		line, err := c.read.ReadString('\n')
		if err != nil {
			return fmt.Errorf("quickaccess: read websocket upgrade header: %w", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}
	return nil
}

func (c *Client) handshake() error {
	encrypted, err := encryptAES([]byte(question), []byte(keyA))
	if err != nil {
		return fmt.Errorf("quickaccess: encrypt the handshaking question: %w", err)
	}

	err = c.writeJSON(map[string]any{
		"PacketType": packetAuth,
		"Version":    1,
		"Session":    sessionID(),
		"Command":    "AuthKey",
		"Param1":     base64.StdEncoding.EncodeToString(encrypted),
	})
	if err != nil {
		return fmt.Errorf("quickaccess: write auth key: %w", err)
	}

	answer, err := c.readJSON()
	if err != nil {
		return fmt.Errorf("quickaccess: read auth response: %w", err)
	}
	data, _ := answer["Data"].(string)
	expected, err := encryptAES([]byte(question), []byte(keyB))
	if err != nil {
		return err
	}
	if data != base64.StdEncoding.EncodeToString(expected) {
		return fmt.Errorf("quickaccess: the service answered the authentication with an unknown question")
	}
	return nil
}

// Close 关闭与 QuickAccess 服务的连接。
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.read = nil
	return err
}

// Get 向服务发送获取指定命令参数的请求。
func (c *Client) Get(command string) (Result, error) {
	err := c.writeJSON(map[string]any{
		"PacketType": packetRequest,
		"Version":    1,
		"Session":    sessionID(),
		"Command":    command,
		"Action":     ActionGet,
	})
	if err != nil {
		return Result{}, err
	}

	deadline := c.deadline()
	for time.Now().Before(deadline) {
		c.conn.SetReadDeadline(deadline)

		answer, err := c.readJSON()
		if err != nil {
			return Result{}, fmt.Errorf("quickaccess: %s: %w", command, err)
		}

		result, matched, err := resultOf(answer, command)
		if err != nil {
			return Result{}, err
		}
		if matched {
			return result, nil
		}
	}
	return Result{}, fmt.Errorf("quickaccess: %s: 服务没有回答这个请求", command)
}

// Set 向服务发送设置命令及新值。
func (c *Client) Set(command string, value any) error {
	err := c.writeJSON(map[string]any{
		"PacketType": packetRequest,
		"Version":    1,
		"Session":    sessionID(),
		"Command":    command,
		"Action":     ActionSet,
		"Param1":     value,
	})
	if err != nil {
		return err
	}

	deadline := c.deadline()
	for time.Now().Before(deadline) {
		c.conn.SetReadDeadline(deadline)

		answer, err := c.readJSON()
		if err != nil {
			return fmt.Errorf("quickaccess: %s: %w", command, err)
		}
		if answersCommand(answer, command) {
			return nil
		}
	}
	return fmt.Errorf("quickaccess: %s: 服务没有回答这个写入", command)
}

func (c *Client) deadline() time.Time {
	timeout := c.timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return time.Now().Add(timeout)
}

func answersCommand(answer map[string]any, command string) bool {
	if kind, _ := answer["PacketType"].(float64); int(kind) != packetResponse {
		return false
	}
	name, _ := answer["Command"].(string)
	return name == "" || name == command
}

// Result 承载 QuickAccess 查询返回的值与描述信息。
type Result struct {
	Value       any
	Text        string
	Description string
}

func resultOf(answer map[string]any, command string) (Result, bool, error) {
	if !answersCommand(answer, command) {
		return Result{}, false, nil
	}
	if action, _ := answer["Action"].(string); action != "" && action != ActionGet {
		return Result{}, false, nil
	}

	result := Result{}
	payload, ok := answer["Result"]
	if !ok {
		return result, true, nil
	}
	switch typed := payload.(type) {
	case map[string]any:
		result.Value = typed["Value"]
		result.Text, _ = typed["Text"].(string)
		result.Description, _ = typed["Description"].(string)
	case []any:
		for _, entry := range typed {
			entryMap, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			text, _ := entryMap["Text"].(string)
			description, _ := entryMap["Description"].(string)
			if description != "" {
				result.Description = description
			}
			if text != "" {
				result.Text = text
			}
			result.Value = entryMap["Value"]
		}
	default:
		result.Value = payload
	}
	return result, true, nil
}

const (
	opcodeText  = 1
	opcodeClose = 8
	opcodePing  = 9
	opcodePong  = 10
)

func (c *Client) writeJSON(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.writeFrame(opcodeText, payload)
}

func (c *Client) writeFrame(opcode byte, payload []byte) error {
	length := len(payload)
	frame := make([]byte, 0, 14+length)
	frame = append(frame, 0x80|(opcode&0x0F))

	switch {
	case length <= 125:
		frame = append(frame, 0x80|byte(length))
	case length <= 65535:
		frame = append(frame, 0x80|126, byte(length>>8), byte(length))
	default:
		frame = append(frame, 0x80|127)
		var extended [8]byte
		binary.BigEndian.PutUint64(extended[:], uint64(length))
		frame = append(frame, extended[:]...)
	}

	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return err
	}
	frame = append(frame, mask[:]...)
	for i, b := range payload {
		frame = append(frame, b^mask[i%4])
	}

	if c.timeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	}
	_, err := c.conn.Write(frame)
	return err
}

func (c *Client) readJSON() (map[string]any, error) {
	var message []byte
	for {
		opcode, payload, err := c.readFrame()
		if err != nil {
			return nil, err
		}
		switch opcode {
		case opcodeText:
			message = payload
		case opcodePing:
			if err := c.writeFrame(opcodePong, payload); err != nil {
				return nil, err
			}
			continue
		case opcodePong:
			continue
		case opcodeClose:
			return nil, fmt.Errorf("quickaccess: the service closed the session")
		default:
			continue
		}
		if len(message) == 0 {
			continue
		}
		break
	}

	var answer map[string]any
	if err := json.Unmarshal(message, &answer); err != nil {
		return nil, fmt.Errorf("quickaccess: cannot read the answer %q: %w", string(message), err)
	}
	return answer, nil
}

func (c *Client) readFrame() (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.read, header); err != nil {
		return 0, nil, err
	}

	opcode := header[0] & 0x0F
	masked := header[1]&0x80 != 0
	length := int(header[1] & 0x7F)

	switch length {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(c.read, extended); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(c.read, extended); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint64(extended))
	}

	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(c.read, mask[:]); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(c.read, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return opcode, payload, nil
}

func encryptAES(plain []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padding := aes.BlockSize - len(plain)%aes.BlockSize
	padded := make([]byte, 0, len(plain)+padding)
	padded = append(padded, plain...)
	for i := 0; i < padding; i++ {
		padded = append(padded, byte(padding))
	}

	out := make([]byte, len(padded))
	for offset := 0; offset < len(padded); offset += aes.BlockSize {
		block.Encrypt(out[offset:offset+aes.BlockSize], padded[offset:offset+aes.BlockSize])
	}
	return out, nil
}

func sessionID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	raw[6] = (raw[6] & 0x0F) | 0x40
	raw[8] = (raw[8] & 0x3F) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}
