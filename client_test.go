package http

import (
	"io"
	"net/http"
	"testing"

	"github.com/gorilla/http/client"
)

var clientDoTests = []struct {
	// arguments
	method, path string
	headers      map[string][]string
	body         io.Reader
	// return values
	client.Status
	rheaders map[string][]string
	rbody    io.Reader
	err      error
}{
	{method: "GET",
		path:   "/200",
		Status: client.Status{200, "OK"},
	},
	{method: "GET",
		path:   "/404",
		Status: client.Status{404, "Not Found"},
	},
}

func TestClientDo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/200", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("OK")) })
	s := newServer(t, mux)
	defer s.Shutdown()
	for _, tt := range clientDoTests {
		var c Client
		url := s.Root() + tt.path
		status, _, _, err := c.Do(tt.method, url, tt.headers, tt.body)
		if err != tt.err {
			t.Errorf("Client.Do(%q, %q, %v, %v): err expected %v, got %v", tt.method, tt.path, tt.headers, tt.body, tt.err, err)
		}
		if status != tt.Status {
			t.Errorf("Client.Do(%q, %q, %v, %v): status expected %v, got %v", tt.method, tt.path, tt.headers, tt.body, tt.Status, status)
		}
	}
}

func TestDefaultClientDo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/200", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("OK")) })
	s := newServer(t, mux)
	defer s.Shutdown()
	for _, tt := range clientDoTests {
		url := s.Root() + tt.path
		status, _, _, err := DefaultClient.Do(tt.method, url, tt.headers, tt.body)
		if err != tt.err {
			t.Errorf("Client.Do(%q, %q, %v, %v): err expected %v, got %v", tt.method, tt.path, tt.headers, tt.body, tt.err, err)
		}
		if status != tt.Status {
			t.Errorf("Client.Do(%q, %q, %v, %v): status expected %v, got %v", tt.method, tt.path, tt.headers, tt.body, tt.Status, status)
		}
	}
}

var clientGetTests = []struct {
	path    string
	headers map[string][]string
	client.Status
	rheaders map[string][]string
	rbody    io.Reader
	err      error
}{
	{
		path:   "/200",
		Status: client.Status{200, "OK"},
	},
	{
		path:   "/404",
		Status: client.Status{404, "Not Found"},
	},
}

func TestClientGet(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/200", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("OK")) })
	s := newServer(t, mux)
	defer s.Shutdown()
	for _, tt := range clientGetTests {
		var c Client
		url := s.Root() + tt.path
		status, _, _, err := c.Get(url, tt.headers)
		if err != tt.err {
			t.Errorf("Client.Get(%q, %v): err expected %v, got %v", tt.path, tt.headers, tt.err, err)
		}
		if status != tt.Status {
			t.Errorf("Client.Get(%q, %v): status expected %v, got %v", tt.path, tt.headers, tt.Status, status)
		}
	}
}
