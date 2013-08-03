package http

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gorilla/http/client"
)

const postBody = "banananana"

var clientDoTests = []struct {
	Client
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

		Client:   Client{dialer: new(dialer)},
		method:   "GET",
		path:     "/200",
		Status:   client.Status{200, "OK"},
		rheaders: map[string][]string{"Content-Length": []string{"2"}, "Content-Type": []string{"text/plain; charset=utf-8"}},
		rbody:    strings.NewReader("OK"),
	},
	{
		Client:   Client{dialer: new(dialer)},
		method:   "GET",
		path:     "/404",
		Status:   client.Status{404, "Not Found"},
		rheaders: map[string][]string{"Content-Length": []string{"19"}, "Content-Type": []string{"text/plain; charset=utf-8"}},
		rbody:    strings.NewReader("404 page not found\n"),
	},
	{
		Client:   Client{dialer: new(dialer)},
		method:   "GET",
		path:     "/a",
		Status:   client.Status{200, "OK"},
		rheaders: map[string][]string{"Transfer-Encoding": {"chunked"}, "Content-Type": []string{"text/plain; charset=utf-8"}},
		rbody:    strings.NewReader(a()),
	},
	{
		Client:  Client{dialer: new(dialer)},
		method:  "GET",
		path:    "/a",
		Status:  client.Status{200, "OK"},
		headers: map[string][]string{"Accept-Encoding": []string{"gzip"}},
		rheaders: map[string][]string{
			// net/http can buffer the first write to avoid chunked
			"Content-Length":   []string{"48"},
			"Content-Encoding": []string{"gzip"},
			"Content-Type":     []string{"application/x-gzip"},
		},
		rbody: strings.NewReader(a()),
	},
	{
		Client:   Client{dialer: new(dialer)},
		method:   "POST",
		path:     "/201",
		body:     func() io.Reader { return strings.NewReader(postBody) },
		Status:   client.Status{201, "Created"},
		rheaders: map[string][]string{"Content-Length": []string{"8"}, "Content-Type": []string{"text/plain; charset=utf-8"}},
		rbody:    strings.NewReader("Created\n"),
	},
	{
		Client: Client{dialer: new(dialer)},
		method: "GET",
		path:   "/301",
		Status: client.Status{301, "Moved Permanently"},
		rheaders: map[string][]string{
			"Location":       []string{"/200"},
			"Content-Length": []string{"39"},
			"Content-Type":   []string{"text/html; charset=utf-8"},
		},
		rbody: strings.NewReader("<a href=\"/200\">Moved Permanently</a>.\n\n"),
	},
	{
		Client: Client{dialer: new(dialer)},
		method: "GET",
		path:   "/302",
		Status: client.Status{302, "Found"},
		rheaders: map[string][]string{
			"Location":       []string{"/200"},
			"Content-Length": []string{"27"},
			"Content-Type":   []string{"text/html; charset=utf-8"},
		},
		rbody: strings.NewReader("<a href=\"/200\">Found</a>.\n\n"),
	},
}

func stdmux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/200", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("OK")) })
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		var ww io.Writer = w
		if r.Header.Get("Accept-Encoding") == "gzip" {
			w.Header().Add("Content-Encoding", "gzip")
			ww = gzip.NewWriter(ww)
		}
		for i := 0; i < 1024; i++ {
			ww.Write([]byte("aaaaaaaa"))
		}
		if w, ok := ww.(*gzip.Writer); ok {
			w.Close()
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
	mux.HandleFunc("/301", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "200", 301)
	})
	mux.HandleFunc("/302", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "200", 302)
	})
	return mux
}

func TestClientDo(t *testing.T) {
	s := newServer(t, stdmux())
	defer s.Shutdown()
	for _, tt := range clientDoTests {
		url := s.Root() + tt.path
		var body io.Reader
		if tt.body != nil {
			body = tt.body()
		}
		status, headers, rbody, err := tt.Client.Do(tt.method, url, tt.headers, body)
		if err != tt.err {
			t.Errorf("Client.Do(%q, %q, %v, %v): err expected %v, got %v", tt.method, tt.path, tt.headers, tt.body, tt.err, err)
		}
		if err != nil {
			continue
		}
		if status != tt.Status {
			t.Errorf("Client.Do(%q, %q, %v, %v): status expected %v, got %v", tt.method, tt.path, tt.headers, tt.body, tt.Status, status)
		}
		delete(headers, "Date") // hard to predict
		if !reflect.DeepEqual(tt.rheaders, headers) {
			t.Errorf("Client.Do(%q, %q, %v, %v): headers expected %v, got %v", tt.method, tt.path, tt.headers, tt.body, tt.rheaders, headers)
		}
		if actual, expected := readBodies(t, rbody, tt.rbody); actual != expected {
			t.Errorf("Client.Do(%q, %q, %v, %v): body expected %q, got %q", tt.method, tt.path, tt.headers, tt.body, expected, actual)
		}
	}
}

func readBodies(t *testing.T, a, b io.Reader) (string, string) {
	return readBody(t, a), readBody(t, b)
}

func readBody(t *testing.T, r io.Reader) string {
	if r == nil {
		return ""
	}
	var b bytes.Buffer
	if _, err := io.Copy(&b, r); err != nil {
		t.Fatal(err)
	}
	return b.String()
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
		c := &Client{dialer: new(dialer)}
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
		c := &Client{dialer: new(dialer)}
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
