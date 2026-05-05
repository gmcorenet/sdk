package gmcore_transport

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type TCPConfig struct {
	Host  string
	Port  int
	Ports []int
}

type TCPServer struct {
	config  TCPConfig
	ln      net.Listener
	sec     SecurityProvider
	handler CommandHandler
	mu      sync.RWMutex
	closed  bool
}

func NewTCPServer(cfg TCPConfig) *TCPServer {
	return &TCPServer{config: cfg}
}

func (s *TCPServer) UseSecurity(sec SecurityProvider) {
	s.sec = sec
}

func (s *TCPServer) SetHandler(h CommandHandler) {
	s.handler = h
}

func (s *TCPServer) Listen(ctx context.Context) error {
	var addr string
	if len(s.config.Ports) > 0 {
		addr = fmt.Sprintf("%s:%d", s.config.Host, s.config.Ports[0])
	} else if s.config.Port > 0 {
		addr = fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	} else {
		addr = fmt.Sprintf("%s:8080", s.config.Host)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on TCP: %w", err)
	}
	s.ln = ln

	return s.serve(ctx)
}

func (s *TCPServer) serve(ctx context.Context) error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			s.mu.RLock()
			closed := s.closed
			s.mu.RUnlock()

			if closed {
				return nil
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		}

		go s.handleConn(conn)
	}
}

func (s *TCPServer) handleConn(conn net.Conn) {
	defer conn.Close()

	if s.sec != nil {
		if err := s.sec.Handshake(conn); err != nil {
			return
		}
	}

	if s.handler != nil {
		s.handleRaw(conn)
	}
}

func (s *TCPServer) handleRaw(conn net.Conn) {
	buf := make([]byte, 64*1024)
	for {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		n, err := conn.Read(buf)
		if n > 0 {
			data := buf[:n]

			if s.sec != nil && s.sec.Type() != SecurityNone {
				if len(data) < 32 {
					continue
				}
				payload := data[:len(data)-32]
				sig := data[len(data)-32:]
				if !s.sec.Verify(payload, sig) {
					conn.Write([]byte("SECURITY_ERROR"))
					continue
				}
				data = payload
			}

			resp, err := s.handler("tcp", data)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("ERROR: %v", err)))
				continue
			}
			conn.Write(resp)
		}
		if err != nil {
			break
		}
	}
}

func (s *TCPServer) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *TCPServer) Addr() string {
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

type TCPClient struct {
	host   string
	port   int
	sec    SecurityProvider
	conn   net.Conn
	mu     sync.Mutex
}

func NewTCPClient(host string, port int) *TCPClient {
	return &TCPClient{host: host, port: port}
}

func (c *TCPClient) UseSecurity(sec SecurityProvider) {
	c.sec = sec
}

func (c *TCPClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to TCP: %w", err)
	}
	c.conn = conn

	if c.sec != nil {
		if err := c.sec.Handshake(c.conn); err != nil {
			c.conn.Close()
			return fmt.Errorf("handshake failed: %w", err)
		}
	}

	return nil
}

func (c *TCPClient) Send(data []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, ErrNotConnected
	}

	if c.sec != nil && c.sec.Type() != SecurityNone {
		data = append(data, c.sec.Sign(data)...)
	}

	_, err := c.conn.Write(data)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 64*1024)
	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

func (c *TCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *TCPClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func ParseHostPort(addr string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}

	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		return "", 0, err
	}

	return host, port, nil
}
