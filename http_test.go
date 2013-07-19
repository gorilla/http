package http_test

import (
	"fmt"
	"net"
	stdhttp "net/http"
	"testing"

	"github.com/gorilla/http"
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
	return fmt.Sprintf("http://%s/", s.Listener.Addr().String())
}

// starts a new net/http http server
func newServer(t *testing.T) *server {
	l, err := net.ListenTCP("tcp4", localhost)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if err := stdhttp.Serve(l, nil); err != nil {
			// t.Error(err)
		}
	}()
	return &server{t, l}
}

func TestInternalHttpServer(t *testing.T) {
	newServer(t).Shutdown()
}

func testGet(t *testing.T) {
	s := newServer(t)
	defer s.Shutdown()
	if _, err := http.Get(s.Root()); err != nil {
		t.Fatal(err)
	}
}
