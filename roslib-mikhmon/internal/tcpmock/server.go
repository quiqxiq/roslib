package tcpmock

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// Server adalah TCP server minimal RouterOS API.
// Single connection, scripted reply.
type Server struct {
	ln net.Listener

	mu        sync.Mutex
	received  [][]string
	scripted  [][]string // urutan FIFO; setelah habis, server idle (drain only)
	closeOnce sync.Once
	conn      net.Conn
	done      chan struct{}
}

// Start membuka listener pada 127.0.0.1:0 dan kembalikan Server siap pakai.
// Goroutine accept dimulai langsung.
func Start() (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("tcpmock: listen: %w", err)
	}
	s := &Server{ln: ln, done: make(chan struct{})}
	go s.accept()
	return s, nil
}

// Addr alamat listener (host:port).
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Script men-stack urutan reply yang akan dikirim ke client.
// Setiap call DecodeSentence sukses akan men-pop satu reply dan EncodeSentence-kan.
func (s *Server) Script(replies ...[]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scripted = append(s.scripted, replies...)
}

// Received mengembalikan snapshot semua sentence yang sudah masuk.
func (s *Server) Received() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]string, len(s.received))
	for i, r := range s.received {
		out[i] = append([]string(nil), r...)
	}
	return out
}

// Close menutup koneksi dan listener. Aman dipanggil berkali-kali.
func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		_ = s.ln.Close()
		s.mu.Lock()
		c := s.conn
		s.mu.Unlock()
		if c != nil {
			_ = c.Close()
		}
		close(s.done)
	})
	return nil
}

func (s *Server) accept() {
	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	defer conn.Close()

	r := bufio.NewReader(conn)
	for {
		words, err := DecodeSentence(r)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			return
		}
		s.mu.Lock()
		s.received = append(s.received, words)
		var reply []string
		if len(s.scripted) > 0 {
			reply = s.scripted[0]
			s.scripted = s.scripted[1:]
		}
		s.mu.Unlock()
		if reply != nil {
			_ = EncodeSentence(conn, reply)
		}
	}
}
