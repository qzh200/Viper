package service

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

// The server answers the official application. The tests below walk
// through the packets of the protocol and check the answers, because a
// client must not be able to tell the two services apart.

func TestAnswerOfTheInitialization(t *testing.T) {
	server := &Server{provider: NewProvider(nil, nil)}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet:   PacketInitialization,
		Function: capVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"data\" : \"1.2.2101.0\",\n\t\"request\" : \"VERSION\",\n\t\"result\" : \"0\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestAnswerOfAnUnknownAbility(t *testing.T) {
	// The original service leaves out the data when it does not know the
	// ability, the result code says that it could not answer.
	server := &Server{provider: NewProvider(nil, nil)}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet:   PacketInitialization,
		Function: "SUPPORT_NOTHING",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"data\" : null,\n\t\"request\" : \"SUPPORT_NOTHING\",\n\t\"result\" : \"3\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestAnswerOfTheConnectedDevices(t *testing.T) {
	server := &Server{provider: NewProvider(nil, nil)}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet: PacketGetConnectedDevice,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !bytes.Contains([]byte(text), []byte("GET_CONNECTED_DEVICE")) {
		t.Errorf("the answer does not name the request:\n%s", text)
	}
	if !bytes.Contains([]byte(text), []byte("AcerECKeyboard Device")) {
		t.Errorf("the answer does not carry the keyboard:\n%s", text)
	}
}

func TestAnswerOfTheLightingEffects(t *testing.T) {
	// The original service does not answer this request, it leaves the
	// answer empty.
	server := &Server{provider: NewProvider(nil, nil)}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet: PacketGetLightingEffect,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"data\" : null,\n\t\"request\" : null,\n\t\"result\" : \"0\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestAnswerOfAnUnknownSetting(t *testing.T) {
	server := &Server{provider: NewProvider(nil, nil)}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet:   PacketGetUpdatedData,
		Function: "NOTHING",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"data\" : null,\n\t\"request\" : \"NOTHING\",\n\t\"result\" : \"3\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestAnswerOfAWriteThatCannotBeDone(t *testing.T) {
	// A setting that cannot be written is answered with the code of the
	// failure, the answer carries no data.
	server := &Server{provider: NewProvider(nil, nil)}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet:   PacketSetDeviceData,
		Function: "NOTHING",
		Parameter: map[string]any{
			"status": float64(1),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"request\" : \"NOTHING\",\n\t\"result\" : \"4\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestAnswerOfAnUnknownPacket(t *testing.T) {
	server := &Server{provider: NewProvider(nil, nil)}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet:   999,
		Function: "VERSION",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"data\" : null,\n\t\"request\" : \"VERSION\",\n\t\"result\" : \"3\"\n}\n"
	if out.String() != want {
		t.Errorf("the answer is\n%q\nwant\n%q", out.String(), want)
	}
}

func TestAnswerOfTheSoundMode(t *testing.T) {
	// The sound mode does not need the gaming provider, so the server can
	// answer it without the rights of an administrator.
	server := &Server{provider: NewProvider(nil, nil)}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet:   PacketGetUpdatedData,
		Function: "SOUND_MODE",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !bytes.Contains([]byte(text), []byte("DTS")) {
		t.Errorf("the answer does not carry the sound chip:\n%s", text)
	}
}

// bufferConn is a connection that keeps everything that is written to it.
type bufferConn struct {
	*bytes.Buffer
}

func (c *bufferConn) Read(p []byte) (int, error) { return 0, io.EOF }

func (c *bufferConn) Close() error { return nil }

func (c *bufferConn) LocalAddr() net.Addr { return nil }

func (c *bufferConn) RemoteAddr() net.Addr { return nil }

func (c *bufferConn) SetDeadline(t time.Time) error { return nil }

func (c *bufferConn) SetReadDeadline(t time.Time) error { return nil }

func (c *bufferConn) SetWriteDeadline(t time.Time) error { return nil }
