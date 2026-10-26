// Package quickaccess talks to the Acer Quick Access service.
//
// The service is part of the original PredatorSense stack. It offers the
// settings that do not belong to the gaming hardware provider, such as the
// blue light shield and the charging over USB.
//
// The service listens on a local port and speaks the WebSocket protocol
// over TLS. The first packet of a session is an authentication packet whose
// payload is sealed with AES in ECB mode, the packets that follow carry a
// command and an action.
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

// The constants of the protocol.
//
// They are the same for every service of the original stack, only the
// question of the handshake differs.
const (
	// defaultPort is the port of the Quick Access service.
	defaultPort = 5141

	// packetAuth is the type of the authentication packet.
	packetAuth = 1
	// packetRequest is the type of a request packet.
	packetRequest = 2
	// packetResponse is the type of an answer packet.
	packetResponse = 3

	// keyA seals the authentication packet.
	keyA = "A6052DC8A6E44210"
	// keyB is the key that the service uses for its answer.
	keyB = "AB252AB73BED1CDB"

	// question is the text that both sides encrypt to prove that they
	// share the same keys.
	question = "AcerQuickAccessFunctionalityTests_HandshakingQuestion"
)

// The actions of a request packet.
const (
	// ActionGet asks for the current value of a command.
	ActionGet = "Get"
	// ActionSet changes the value of a command.
	ActionSet = "Set"
)

// Client is a connection to the Quick Access service.
type Client struct {
	conn net.Conn
	read *bufio.Reader
	host string
}

// Options change the behaviour of a client.
type Options struct {
	// Address is the address of the service. The default is the local
	// port of the Quick Access service.
	Address string
	// Timeout limits the time of a single request.
	Timeout time.Duration
}

// Connect opens a session with the Quick Access service.
//
// The session is authenticated right away, a service that does not know
// the shared keys refuses the connection.
func Connect(options Options) (*Client, error) {
	address := options.Address
	if address == "" {
		address = fmt.Sprintf("127.0.0.1:%d", defaultPort)
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
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

	client := &Client{conn: conn, read: bufio.NewReader(conn), host: address}
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

// upgrade turns the connection into a WebSocket session.
//
// The service only speaks the WebSocket protocol, the plain stream is
// refused.
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

	if _, err := c.conn.Write([]byte(request)); err != nil {
		return fmt.Errorf("quickaccess: the handshake failed: %w", err)
	}

	status, err := c.read.ReadString('\n')
	if err != nil {
		return fmt.Errorf("quickaccess: the service did not answer the handshake: %w", err)
	}
	if !strings.Contains(status, "101") {
		return fmt.Errorf("quickaccess: the service refused the handshake: %s", strings.TrimSpace(status))
	}
	for {
		line, err := c.read.ReadString('\n')
		if err != nil {
			return fmt.Errorf("quickaccess: the service did not answer the handshake: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}
	return nil
}

// Close ends the session.
func (c *Client) Close() error { return c.conn.Close() }

// handshake performs the authentication of the session.
func (c *Client) handshake() error {
	payload, err := json.Marshal(map[string]string{
		"Question": question,
		"Key":      keyB,
	})
	if err != nil {
		return err
	}
	sealed, err := encryptAES(payload, []byte(keyA))
	if err != nil {
		return err
	}

	err = c.writeJSON(map[string]any{
		"PacketType": packetAuth,
		"Session":    sessionID(),
		"Version":    1,
		"Data":       base64.StdEncoding.EncodeToString(sealed),
	})
	if err != nil {
		return err
	}

	answer, err := c.readJSON()
	if err != nil {
		return fmt.Errorf("quickaccess: the service did not answer the authentication: %w", err)
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

// Get asks for the current value of a command.
//
// The answer carries the value in the field Result, the service describes
// the value in the fields Text and Description.
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

	answer, err := c.readJSON()
	if err != nil {
		return Result{}, fmt.Errorf("quickaccess: %s: %w", command, err)
	}
	result, ok, err := resultOf(answer, command)
	if err != nil {
		return Result{}, err
	}
	if !ok {
		return Result{}, fmt.Errorf("quickaccess: %s: the service did not answer the request", command)
	}
	return result, nil
}

// Set changes the value of a command.
//
// The service answers with a notification first and repeats the new value
// afterwards, the answer that carries the value ends the request.
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

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		answer, err := c.readJSON()
		if err != nil {
			return fmt.Errorf("quickaccess: %s: %w", command, err)
		}
		if answersCommand(answer, command) {
			return nil
		}
	}
	return nil
}

// answersCommand reports whether an answer belongs to a command.
//
// The service answers a request with the packet type of a response and
// the name of the command, the action tells what kind of answer it is.
func answersCommand(answer map[string]any, command string) bool {
	if kind, _ := answer["PacketType"].(float64); int(kind) != packetResponse {
		return false
	}
	name, _ := answer["Command"].(string)
	return name == "" || name == command
}

// Result is the value of a command.
type Result struct {
	// Value is the raw value of the command.
	Value any
	// Text describes the value.
	Text string
	// Description describes the value.
	Description string
}

// resultOf reads the answer of a command.
//
// The second result is false when the answer belongs to another command,
// the service sends notifications of its own beside the answers.
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
		// The answer of a command that reports several values carries the
		// one that is marked as the current value.
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

// writeJSON sends one text frame.
func (c *Client) writeJSON(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.writeFrame(opcodeText, payload)
}

// writeFrame sends one frame.
func (c *Client) writeFrame(opcode byte, payload []byte) error {
	frame := []byte{0x80 | opcode} // the last frame of a message
	switch length := len(payload); {
	case length < 126:
		frame = append(frame, byte(0x80|length))
	case length < 65536:
		frame = append(frame, 0x80|126, byte(length>>8), byte(length))
	default:
		frame = append(frame, 0x80|127)
		var extended [8]byte
		binary.BigEndian.PutUint64(extended[:], uint64(length))
		frame = append(frame, extended[:]...)
	}

	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	frame = append(frame, mask...)
	for i, b := range payload {
		frame = append(frame, b^mask[i%4])
	}

	_, err := c.conn.Write(frame)
	return err
}

// The opcodes of the WebSocket protocol.
const (
	opcodeContinuation = 0x0
	opcodeText         = 0x1
	opcodeBinary       = 0x2
	opcodeClose        = 0x8
	opcodePing         = 0x9
	opcodePong         = 0xA
)

// readJSON reads one text message.
func (c *Client) readJSON() (map[string]any, error) {
	var message []byte
	for {
		opcode, payload, err := c.readFrame()
		if err != nil {
			return nil, err
		}
		switch opcode {
		case opcodeText, opcodeBinary, opcodeContinuation:
			message = append(message, payload...)
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

// readFrame reads one frame.
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

// encryptAES seals a text with AES in ECB mode.
//
// The padding is the one of the original service, every block that is not
// full is filled up to the block size.
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

// sessionID builds the identifier of a packet.
func sessionID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	raw[6] = (raw[6] & 0x0F) | 0x40
	raw[8] = (raw[8] & 0x3F) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}
