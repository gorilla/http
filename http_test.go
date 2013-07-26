package http

import (
	"fmt"
	"net"
	"net/http"
	"testing"
)

var localhost = &net.TCPAddr{
	IP:   net.IP{127, 0, 0, 1},
	Port: 0, // os assigned
}

type server struct {
	*testing.T
	net.Listener
}

// Shutdown should be called to terminate this server.
func (s *server) Shutdown() {
	s.Listener.Close()
}

// Root returns a http URL for the root of this server.
func (s *server) Root() string {
	return fmt.Sprintf("http://%s", s.Listener.Addr().String())
}

// starts a new net/http http server
func newServer(t *testing.T, mux *http.ServeMux) *server {
	l, err := net.ListenTCP("tcp4", localhost)
	if err != nil {
		t.Fatal(err)
	}
	// /404 is not handled, generating a 404
	go func() {
		if err := http.Serve(l, mux); err != nil {
			// t.Error(err)
		}
	}()
	return &server{t, l}
}

func TestInternalHttpServer(t *testing.T) {
	newServer(t, nil).Shutdown()
}

var getTests = []struct {
	path     string
	expected string
	err      error
}{
	{"/200", "OK", nil},
}

func TestGet(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/200", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("OK")) })
	s := newServer(t, mux)
	defer s.Shutdown()
	for _, tt := range getTests {
		url := s.Root() + tt.path
		actual, err := Get(url)
		if actual := string(actual); actual != tt.expected || err != tt.err {
			t.Errorf("Get(%q): expected %q %v, got %q %v", tt.path, tt.expected, tt.err, actual, err)
		}
	}
}
