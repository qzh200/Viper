package service

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

// The server offers the service of the original software on a local
// port. The tests below drive it over a real connection, so that the
// whole path from the request to the answer is covered.

func TestTheServerAnswersOverThePort(t *testing.T) {
	server, err := New(Options{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	go server.Serve()

	conn, err := net.DialTimeout("tcp", server.Address(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// The request of the version.
	frame := []byte("ACER")
	frame = append(frame, 0, 0, 0, 0)
	frame = append(frame, `{"Function":"VERSION"}`...)
	if _, err := conn.Write(frame); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	answer := make([]byte, 4096)
	n, err := conn.Read(answer)
	if err != nil {
		t.Fatal(err)
	}

	var document map[string]any
	if err := json.Unmarshal(answer[:n], &document); err != nil {
		t.Fatalf("the answer cannot be read: %v\n%s", err, answer[:n])
	}
	if document["request"] != "VERSION" {
		t.Errorf("the answer is about %v, want VERSION", document["request"])
	}
	if document["result"] != "0" {
		t.Errorf("the result is %v, want 0", document["result"])
	}
	if document["data"] != ServiceVersion {
		t.Errorf("the version is %v, want %s", document["data"], ServiceVersion)
	}
}

func TestTheServerAnswersTheDevices(t *testing.T) {
	server, err := New(Options{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	go server.Serve()

	conn, err := net.DialTimeout("tcp", server.Address(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	frame := []byte("ACER")
	frame = append(frame, 30, 0, 0, 0)
	frame = append(frame, `{"Function":"GET_CONNECTED_DEVICE"}`...)
	if _, err := conn.Write(frame); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	answer := make([]byte, 8192)
	n, err := conn.Read(answer)
	if err != nil {
		t.Fatal(err)
	}

	var document map[string]any
	if err := json.Unmarshal(answer[:n], &document); err != nil {
		t.Fatalf("the answer cannot be read: %v\n%s", err, answer[:n])
	}
	if document["request"] != "GET_CONNECTED_DEVICE" {
		t.Errorf("the answer is about %v, want GET_CONNECTED_DEVICE", document["request"])
	}
}

func TestTheServerClosesTheSessionOfACleanClient(t *testing.T) {
	server, err := New(Options{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	go server.Serve()

	conn, err := net.DialTimeout("tcp", server.Address(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	// A client that closes the connection without a request must not
	// make the server complain.
	conn.Close()

	// The server is still able to answer the next client.
	next, err := net.DialTimeout("tcp", server.Address(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()

	frame := []byte("ACER")
	frame = append(frame, 0, 0, 0, 0)
	frame = append(frame, `{"Function":"VERSION"}`...)
	if _, err := next.Write(frame); err != nil {
		t.Fatal(err)
	}

	next.SetReadDeadline(time.Now().Add(5 * time.Second))
	answer := make([]byte, 4096)
	if _, err := next.Read(answer); err != nil {
		t.Fatal(err)
	}
}

func TestTheServerRefusesThePortThatIsTaken(t *testing.T) {
	first, err := New(Options{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	// The second server wants the same port.
	_, err = New(Options{Address: first.Address()})
	if err == nil {
		t.Error("the port that is taken was opened twice")
	}
}

func TestTheServerIsClosedTwice(t *testing.T) {
	server, err := New(Options{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Errorf("the server was closed twice: %v", err)
	}
}

func TestTheAddressOfTheServer(t *testing.T) {
	server, err := New(Options{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	if server.Address() == "" {
		t.Error("the server has no address")
	}
}
