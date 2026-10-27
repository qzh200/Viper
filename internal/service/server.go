package service

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// ServiceVersion is the version the service reports.
//
// The official application only writes it into its log, the shape of it
// stays the same so that nothing breaks.
const ServiceVersion = "1.2.2101.0"

// DefaultPort is the port the official service listens on.
const DefaultPort = 46933

// Server offers the service of the original software.
//
// The official application connects to the port, so whoever holds the port
// is the service it talks to.
type Server struct {
	listener net.Listener
	provider *Provider

	mutex    sync.Mutex
	closed   bool
	shutdown chan struct{}
}

// Options change the behaviour of a server.
type Options struct {
	// Address is the address the server listens on. The default is the
	// port of the official service.
	Address string
	// Provider gives the numbers the server reports.
	Provider *Provider
}

// New starts a server.
//
// The caller has to close it, the server stops with the machine.
func New(options Options) (*Server, error) {
	address := options.Address
	if address == "" {
		address = fmt.Sprintf("127.0.0.1:%d", DefaultPort)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("service: 端口打不开: %w", err)
	}

	provider := options.Provider
	if provider == nil {
		provider = NewProvider(nil, nil)
	}

	// The load of the processor is a difference between two readings, so
	// the first one is taken right away.
	primeCPUUsage()

	return &Server{
		listener: listener,
		provider: provider,
		shutdown: make(chan struct{}),
	}, nil
}

// Address returns the address the server listens on.
func (s *Server) Address() string { return s.listener.Addr().String() }

// Serve answers the requests until the server is closed.
func (s *Server) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mutex.Lock()
			closed := s.closed
			s.mutex.Unlock()
			if closed {
				return nil
			}
			return err
		}
		go s.handle(conn)
	}
}

// Close stops the server.
func (s *Server) Close() error {
	s.mutex.Lock()
	if s.closed {
		s.mutex.Unlock()
		return nil
	}
	s.closed = true
	close(s.shutdown)
	s.mutex.Unlock()
	return s.listener.Close()
}

// handle answers the requests of one client.
func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	for {
		select {
		case <-s.shutdown:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		request, err := readRequest(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("service: 请求读取失败: %v", err)
			}
			return
		}

		if err := s.answer(conn, request); err != nil {
			log.Printf("service: 应答写入失败: %v", err)
			return
		}
	}
}

// answer writes the answer to one request.
func (s *Server) answer(conn net.Conn, request Request) error {
	switch request.Packet {
	case PacketInitialization:
		// The original service does not implement the overclocking ability,
		// its answer carries no data at all, only the code of the failure.
		if request.Function == capOverclockingModeCapability {
			return writeUnsupportedResponse(conn, request.Function)
		}
		data, ok := s.provider.Capability(request.Function)
		if !ok {
			// The original service leaves out the data when it does not
			// know the ability.
			return writeError(conn, request.Function, "3")
		}
		return writeResponse(conn, request.Function, data)

	case PacketGetMonitorData:
		return writeResponse(conn, "GET_MONITOR_DATA", s.provider.MonitorData())

	case PacketGetUpdatedData:
		data, ok := s.provider.Setting(request.Function)
		if !ok {
			return writeError(conn, request.Function, "3")
		}
		return writeResponse(conn, request.Function, data)

	case PacketGetConnectedDevice:
		return writeResponse(conn, "GET_CONNECTED_DEVICE", s.provider.Devices())

	case PacketGetLightingEffect:
		// The original service does not answer this request, it leaves
		// the answer empty.
		return writeEmptyResponse(conn)

	case PacketSetDeviceData:
		if err := s.provider.Apply(request.Function, request.Parameter); err != nil {
			log.Printf("service: 设置 %s 写入失败: %v", request.Function, err)
			return writeSetError(conn, request.Function, "4")
		}
		return writeSetResponse(conn, request.Function)

	default:
		return writeError(conn, request.Function, "3")
	}
}
