package http

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
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
		path:     "/query1?a=1",
		Status:   client.Status{200, "OK"},
		rheaders: map[string][]string{"Content-Length": []string{"3"}, "Content-Type": []string{"text/plain; charset=utf-8"}},
		rbody:    strings.NewReader("a=1"),
	},
	/** {

	        Client:   Client{dialer: new(dialer)},
	        method:   "GET",
	        path:     "/query1?a=1#ignored", // fragment should be ignored
	        Status:   client.Status{200, "OK"},
	        rheaders: map[string][]string{"Content-Length": []string{"3"}, "Content-Type": []string{"text/plain; charset=utf-8"}},
	        rbody:    strings.NewReader("a=1"),
	},     **/
	{

		Client:   Client{dialer: new(dialer)},
		method:   "GET",
		path:     "/query2?a=1&b=2",
		Status:   client.Status{200, "OK"},
		rheaders: map[string][]string{"Content-Length": []string{"7"}, "Content-Type": []string{"text/plain; charset=utf-8"}},
		rbody:    strings.NewReader("a=1&b=2"),
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
	{
		Client:   Client{dialer: new(dialer), FollowRedirects: true},
		method:   "GET",
		path:     "/301",
		Status:   client.Status{200, "OK"},
		rheaders: map[string][]string{"Content-Length": []string{"2"}, "Content-Type": []string{"text/plain; charset=utf-8"}},
		rbody:    strings.NewReader("OK"),
	},
	{
		Client:   Client{dialer: new(dialer), FollowRedirects: true},
		method:   "GET",
		path:     "/302",
		Status:   client.Status{200, "OK"},
		rheaders: map[string][]string{"Content-Length": []string{"2"}, "Content-Type": []string{"text/plain; charset=utf-8"}},
		rbody:    strings.NewReader("OK"),
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
	mux.HandleFunc("/query1", func(w http.ResponseWriter, r *http.Request) {
		rq := r.URL.RawQuery
		if rq != "a=1" {
			http.Error(w, "Bad Request", 400)
			return
		}
		w.Write([]byte(rq))
	})
	mux.HandleFunc("/query2", func(w http.ResponseWriter, r *http.Request) {
		rq := r.URL.RawQuery
		if rq != "a=1&b=2" {
			http.Error(w, "Bad Request", 400)
			return
		}
		w.Write([]byte(rq))
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
		delete(headers, "Date")                   // hard to predict
		delete(headers, "X-Content-Type-Options") // a free gift from the Go http server
		for _, v := range headers {
			sort.Strings(v)
		}
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

var clientPutTests = []struct {
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

func TestClientPut(t *testing.T) {
	s := newServer(t, stdmux())
	defer s.Shutdown()
	for _, tt := range clientPutTests {
		c := &Client{dialer: new(dialer)}
		url := s.Root() + tt.path
		var body io.Reader
		if tt.body != nil {
			body = tt.body()
		}
		status, _, _, err := c.Put(url, tt.headers, body)
		if err != tt.err {
			t.Errorf("Client.Put(%q, %v): err expected %v, got %v", tt.path, tt.headers, tt.err, err)
		}
		if status != tt.Status {
			t.Errorf("Client.Put(%q, %v): status expected %v, got %v", tt.path, tt.headers, tt.Status, status)
		}
	}
}

var clientPatchTests = []struct {
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

func TestClientPatch(t *testing.T) {
	s := newServer(t, stdmux())
	defer s.Shutdown()
	for _, tt := range clientPatchTests {
		c := &Client{dialer: new(dialer)}
		url := s.Root() + tt.path
		var body io.Reader
		if tt.body != nil {
			body = tt.body()
		}
		status, _, _, err := c.Patch(url, tt.headers, body)
		if err != tt.err {
			t.Errorf("Client.Patch(%q, %v): err expected %v, got %v", tt.path, tt.headers, tt.err, err)
		}
		if status != tt.Status {
			t.Errorf("Client.Patch(%q, %v): status expected %v, got %v", tt.path, tt.headers, tt.Status, status)
		}
	}
}

var clientDeleteTests = []struct {
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

func TestClientDelete(t *testing.T) {
	s := newServer(t, stdmux())
	defer s.Shutdown()
	for _, tt := range clientDeleteTests {
		c := &Client{dialer: new(dialer)}
		url := s.Root() + tt.path
		status, _, _, err := c.Delete(url, tt.headers)
		if err != tt.err {
			t.Errorf("Client.Delete(%q, %v): err expected %v, got %v", tt.path, tt.headers, tt.err, err)
		}
		if status != tt.Status {
			t.Errorf("Client.Delete(%q, %v): status expected %v, got %v", tt.path, tt.headers, tt.Status, status)
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

var toRequestTests = []struct {
	method, path string
	query        []string
	headers      map[string][]string
	*client.Request
}{
	{
		"GET", "/", nil, nil, &client.Request{
			Method:  "GET",
			Path:    "/",
			Version: client.HTTP_1_1,
		},
	},
}

func TestToRequest(t *testing.T) {
	for _, tt := range toRequestTests {
		actual := toRequest(tt.method, tt.path, tt.query, tt.headers, nil) // don't check body
		if !sameRequest(tt.Request, actual) {
			t.Errorf("toRequest(%q, %q, %q, %q, %q): expected %q, got %q", tt.method, tt.path, tt.query, tt.Version, tt.headers, tt.Request, actual)
		}
	}
}

func sameRequest(a, b *client.Request) bool {
	if a.Method != b.Method {
		return false
	}
	if a.Path != b.Path {
		return false
	}
	if !reflect.DeepEqual(a.Query, b.Query) {
		return false
	}
	if a.Version != b.Version {
		return false
	}
	return reflect.DeepEqual(a.Headers, b.Headers)
}

var fromResponseTests = []struct {
	*client.Response
	client.Version
	client.Status
	body    io.Reader
	headers map[string][]string
}{
// TODO(dfc)
}

func TestFromResponse(t *testing.T) {
	for _, tt := range fromResponseTests {
		version, status, headers, body := fromResponse(tt.Response)
		if version != tt.Version {
			t.Errorf("fromRequest(%q): version: expected %v, got %v", tt.Response, tt.Version, version)
		}
		if status != tt.Status {
			t.Errorf("fromRequest(%q): status: expected %v, got %v", tt.Response, tt.Status, status)
		}
		if !reflect.DeepEqual(headers, tt.headers) {
			t.Errorf("fromRequest(%q): headers: expected %v, got %v", tt.Response, tt.headers, headers)
		}
		if same, actual, expected := sameBody(t, body, tt.body); !same {
			t.Errorf("fromRequest(%q): body: expected %q, got %q", tt.Response, expected, actual)
		}
	}
}

// sameBody consumes both bodies.
func sameBody(t *testing.T, a, b io.Reader) (bool, string, string) {
	var A, B bytes.Buffer
	if _, err := io.Copy(&A, a); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(&B, b); err != nil {
		t.Fatal(err)
	}
	return bytes.Equal(A.Bytes(), B.Bytes()), A.String(), B.String()
}

var headerValueTests = []struct {
	headers       map[string][]string
	key, expected string
}{
	{
		key:      "foo",
		expected: "",
	},
	{
		headers:  make(map[string][]string),
		key:      "foo",
		expected: "",
	},
	{
		headers: map[string][]string{
			"foo": nil,
		},
		key:      "foo",
		expected: "",
	},
	{
		headers: map[string][]string{
			"bar": []string{"baz"},
		},
		key:      "foo",
		expected: "",
	},
	{
		headers: map[string][]string{
			"foo": []string{"baz"},
		},
		key:      "foo",
		expected: "baz",
	},
	{
		headers: map[string][]string{
			"foo": []string{"baz", "quzz"},
		},
		key:      "foo",
		expected: "baz quzz",
	},
	{
		headers: map[string][]string{
			"foo": []string{"baz", ""},
		},
		key:      "foo",
		expected: "baz ", // odd
	},
}

func TestHeaderValue(t *testing.T) {
	for _, tt := range headerValueTests {
		actual := headerValue(tt.headers, tt.key)
		if actual != tt.expected {
			t.Errorf("headerValue(%v, %q): expected %q, got %q", tt.headers, tt.key, tt.expected, actual)
		}
	}
}

var firstErrTests = []struct {
	err1, err2 error
	expected   error
}{
	{nil, nil, nil},
	{io.EOF, nil, io.EOF},
	{nil, io.EOF, io.EOF},
	{io.EOF, errors.New("yowzer"), io.EOF},
}

func TestFirstErr(t *testing.T) {
	for _, tt := range firstErrTests {
		actual := firstErr(tt.err1, tt.err2)
		if !sameErr(actual, tt.expected) {
			t.Errorf("firstErr(%v, %v): expected %v, got %v", tt.err1, tt.err2, tt.expected, actual)
		}
	}
}
