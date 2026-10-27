package quickaccess

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"
)

// The service of the original stack speaks the WebSocket protocol over
// TLS. The server below stands for it, so that the tests can drive the
// whole session without the service of the vendor.

// testServer is a stand for the Quick Access service.
type testServer struct {
	// listener accepts the connections.
	listener net.Listener
	// answers holds the answer of every command, keyed by the name of
	// the command.
	answers map[string]map[string]any
	// requests records the requests that were received.
	requests []map[string]any
	// refuseHandshake makes the server refuse the WebSocket handshake.
	refuseHandshake bool
	// wrongQuestion makes the server answer the authentication with
	// something else than the question.
	wrongQuestion bool
	// closeAfterHandshake makes the server close the session.
	closeAfterHandshake bool
}

// newTestServer starts a server on a free port of the loopback.
func newTestServer(t *testing.T) *testServer {
	t.Helper()

	certificate := selfSigned(t)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{certificate},
	})
	if err != nil {
		t.Fatal(err)
	}

	server := &testServer{
		listener: listener,
		answers:  map[string]map[string]any{},
	}
	go server.serve()
	t.Cleanup(func() { listener.Close() })
	return server
}

// address returns the address of the server.
func (s *testServer) address() string { return s.listener.Addr().String() }

// answer sets the answer of a command.
func (s *testServer) answer(command string, value any) *testServer {
	s.answers[command] = map[string]any{"Value": value}
	return s
}

// serve accepts the connections until the listener is closed.
func (s *testServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

// handle answers one session.
func (s *testServer) handle(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// The WebSocket handshake.
	if _, err := reader.ReadString('\n'); err != nil {
		return
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		if len(line) <= 2 {
			break
		}
	}
	if s.refuseHandshake {
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}
	conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))

	// The first packet of the session is the authentication.
	opcode, payload, err := readServerFrame(reader)
	if err != nil {
		return
	}
	if opcode != opcodeText {
		return
	}
	var auth map[string]any
	if err := json.Unmarshal(payload, &auth); err != nil {
		return
	}
	if s.wrongQuestion {
		conn.Write(serverFrame([]byte(`{"Data":"wrong"}`)))
		return
	}
	expected, err := encryptAES([]byte(question), []byte(keyB))
	if err != nil {
		return
	}
	conn.Write(serverFrame([]byte(fmt.Sprintf(
		`{"PacketType":%d,"Data":"%s"}`,
		packetResponse,
		base64.StdEncoding.EncodeToString(expected),
	))))

	if s.closeAfterHandshake {
		return
	}

	// The requests that follow.
	for {
		opcode, payload, err := readServerFrame(reader)
		if err != nil {
			return
		}
		if opcode != opcodeText {
			continue
		}
		var request map[string]any
		if err := json.Unmarshal(payload, &request); err != nil {
			return
		}
		s.requests = append(s.requests, request)

		command, _ := request["Command"].(string)
		answer, ok := s.answers[command]
		if !ok {
			answer = map[string]any{"Value": nil}
		}
		document, err := json.Marshal(map[string]any{
			"PacketType": packetResponse,
			"Command":    command,
			"Action":     ActionGet,
			"Result":     answer,
		})
		if err != nil {
			return
		}
		conn.Write(serverFrame(document))
	}
}

// readServerFrame reads one frame that the client sent.
//
// The client masks its frames, so the mask has to be taken off.
func readServerFrame(reader *bufio.Reader) (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := reader.Read(header); err != nil {
		return 0, nil, err
	}
	opcode := header[0] & 0x0F
	masked := header[1]&0x80 != 0
	length := int(header[1] & 0x7F)

	switch length {
	case 126:
		extended := make([]byte, 2)
		if _, err := reader.Read(extended); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := reader.Read(extended); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint64(extended))
	}

	var mask [4]byte
	if masked {
		if _, err := reader.Read(mask[:]); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := reader.Read(payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return opcode, payload, nil
}

// serverFrame builds a frame that the service sends.
//
// The service does not mask its frames.
func serverFrame(payload []byte) []byte {
	frame := []byte{0x80 | opcodeText}
	switch length := len(payload); {
	case length < 126:
		frame = append(frame, byte(length))
	case length < 65536:
		frame = append(frame, 126, byte(length>>8), byte(length))
	default:
		frame = append(frame, 127)
		var extended [8]byte
		binary.BigEndian.PutUint64(extended[:], uint64(length))
		frame = append(frame, extended[:]...)
	}
	return append(frame, payload...)
}

// selfSigned builds a certificate for the test server.
func selfSigned(t *testing.T) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}
}
