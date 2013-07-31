package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/http/client"
)

const postBody = "banananana"

var clientDoTests = []struct {
	// arguments
	method, path string
	headers      map[string][]string
	body         func() io.Reader
	// return values
	client.Status
	rheaders map[string][]string
	rbody    io.Reader
	err      error
}{
	{
		method: "GET",
		path:   "/200",
		Status: client.Status{200, "OK"},
	},
	{
		method: "GET",
		path:   "/404",
		Status: client.Status{404, "Not Found"},
	},
	{
		method: "GET",
		path:   "/a",
		Status: client.Status{200, "OK"},
		rbody:  strings.NewReader("a"),
	},
	{
		method: "POST",
		path:   "/201",
		body:   func() io.Reader { return strings.NewReader(postBody) },
		Status: client.Status{201, "Created"},
	},
}

func stdmux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/200", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("OK")) })
	mux.HandleFunc("/a", func(w http.ResponseWriter, _ *http.Request) {
		for i := 0; i < 1024; i++ {
			w.Write([]byte("aaaaaaaa"))
		}
	})
	mux.HandleFunc("/201", func(w http.ResponseWriter, r *http.Request) {
		var b bytes.Buffer
		io.Copy(&b, r.Body)
		if b.String() != postBody {
			http.Error(w, fmt.Sprintf("/201, expected %q, got %q", postBody, b.String()), 400)
		} else {
			http.Error(w, "Created", 201)
		}
	})
	return mux
}

func TestClientDo(t *testing.T) {
	s := newServer(t, stdmux())
	defer s.Shutdown()
	for _, tt := range clientDoTests {
		c := &Client{new(dialer)}
		url := s.Root() + tt.path
		var body io.Reader
		if tt.body != nil {
			body = tt.body()
		}
		status, _, _, err := c.Do(tt.method, url, tt.headers, body)
		if err != tt.err {
			t.Errorf("Client.Do(%q, %q, %v, %v): err expected %v, got %v", tt.method, tt.path, tt.headers, tt.body, tt.err, err)
		}
		if status != tt.Status {
			t.Errorf("Client.Do(%q, %q, %v, %v): status expected %v, got %v", tt.method, tt.path, tt.headers, tt.body, tt.Status, status)
		}

	}
}

func TestDefaultClientDo(t *testing.T) {
	s := newServer(t, stdmux())
	defer s.Shutdown()
	for _, tt := range clientDoTests {
		url := s.Root() + tt.path
		var body io.Reader
		if tt.body != nil {
			body = tt.body()
		}
		status, _, _, err := DefaultClient.Do(tt.method, url, tt.headers, body)
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
	s := newServer(t, stdmux())
	defer s.Shutdown()
	for _, tt := range clientGetTests {
		c := &Client{new(dialer)}
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

var clientPostTests = []struct {
	path    string
	headers map[string][]string
	body    func() io.Reader
	client.Status
	rheaders map[string][]string
	rbody    io.Reader
	err      error
}{
	{
		path:   "/201",
		body:   func() io.Reader { return strings.NewReader(postBody) },
		Status: client.Status{201, "Created"},
	},
	{
		path:   "/404",
		Status: client.Status{404, "Not Found"},
	},
}

func TestClientPost(t *testing.T) {
	s := newServer(t, stdmux())
	defer s.Shutdown()
	for _, tt := range clientPostTests {
		c := &Client{new(dialer)}
		url := s.Root() + tt.path
		var body io.Reader
		if tt.body != nil {
			body = tt.body()
		}
		status, _, _, err := c.Post(url, tt.headers, body)
		if err != tt.err {
			t.Errorf("Client.Post(%q, %v): err expected %v, got %v", tt.path, tt.headers, tt.err, err)
		}
		if status != tt.Status {
			t.Errorf("Client.Post(%q, %v): status expected %v, got %v", tt.path, tt.headers, tt.Status, status)
		}
	}
}

// assert that StatusError is an error.
var _ error = new(StatusError)

var statusErrorTests = []struct {
	client.Status
	err error
}{}

func TestStatusError(t *testing.T) {
	for _, tt := range statusErrorTests {
		err := &StatusError{tt.Status}
		if !sameErr(err, tt.err) {
			t.Errorf("StatusError{%q}: expected %v, got %v", tt.Status, tt.err, err)
		}
	}
}
