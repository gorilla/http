package client

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

var phaseStringTests = []struct {
	phase
	expected string
}{
	{0, "requestline"},
	{1, "headers"},
	{2, "body"},
	{3, "UNKNOWN"},
}

func TestPhaseString(t *testing.T) {
	for _, tt := range phaseStringTests {
		actual := tt.phase.String()
		if actual != tt.expected {
			t.Errorf("phase(%d).String(): expected %q, got %q", tt.phase, tt.expected, actual)
		}
	}
}

func TestPhaseError(t *testing.T) {
	var c writer
	err := c.WriteHeader("Host", "localhost")
	if _, ok := err.(*phaseError); !ok {
		t.Fatalf("expected %T, got %v", new(phaseError), err)
	}
	expected := `phase error: expected headers, got requestline`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}

var writeRequestLineTests = []struct {
	method, path, version string
	query                 []string
	expected              string
}{
	{method: "GET", path: "/foo", version: "HTTP/1.0", expected: "GET /foo HTTP/1.0\r\n"},
	{method: "GET", path: "/foo", query: []string{}, version: "HTTP/1.0", expected: "GET /foo HTTP/1.0\r\n"},
	{method: "GET", path: "/foo", query: []string{"hello=foo"}, version: "HTTP/1.0", expected: "GET /foo?hello=foo HTTP/1.0\r\n"},
	{method: "GET", path: "/foo", query: []string{"hello=foo", "bar=quux"}, version: "HTTP/1.0", expected: "GET /foo?hello=foo&bar=quux HTTP/1.0\r\n"},
}

func TestWriteRequestLine(t *testing.T) {
	for _, tt := range writeRequestLineTests {
		var b bytes.Buffer
		c := &writer{Writer: &b}
		if err := c.WriteRequestLine(tt.method, tt.path, tt.query, tt.version); err != nil {
			t.Fatalf("Conn.WriteRequestLine(%q, %q, %v %q): %v", tt.method, tt.path, tt.query, tt.version, err)
		}
		c.Writer.(*bufio.Writer).Flush()
		if actual := b.String(); actual != tt.expected {
			t.Errorf("Conn.WriteRequestLine(%q, %q, %v, %q): expected %q, got %q", tt.method, tt.path, tt.query, tt.version, tt.expected, actual)
		}
	}
}

func TestDoubleRequestLine(t *testing.T) {
	var b bytes.Buffer
	c := &writer{Writer: &b}
	if err := c.WriteRequestLine("GET", "/hello", nil, "HTTP/0.9"); err != nil {
		t.Fatal(err)
	}
	err := c.WriteRequestLine("GET", "/hello", nil, "HTTP/0.9")
	expected := `phase error: expected requestline, got headers`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}

var writeHeaderTests = []struct {
	key, value string
	expected   string
}{
	{"Host", "localhost", "Host: localhost\r\n"},
}

func TestWriteHeader(t *testing.T) {
	for _, tt := range writeHeaderTests {
		var b bytes.Buffer
		c := &writer{Writer: &b}
		c.StartHeaders()
		if err := c.WriteHeader(tt.key, tt.value); err != nil {
			t.Fatalf("Conn.WriteHeader(%q, %q): %v", tt.key, tt.value, err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("Conn.WriteHeader(%q, %q): expected %q, got %q", tt.key, tt.value, tt.expected, actual)
		}
	}
}

func TestStartBody(t *testing.T) {
	var b bytes.Buffer
	c := &writer{Writer: bufio.NewWriter(&b), tmp: &b}
	c.StartHeaders()
	if err := c.WriteHeader("Host", "localhost"); err != nil {
		t.Fatal(err)
	}
	c.StartBody()
	err := c.WriteHeader("Connection", "close")
	if _, ok := err.(*phaseError); !ok {
		t.Fatalf("expected %T, got %v", new(phaseError), err)
	}
	expected := `phase error: expected headers, got body`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
	expected = "Host: localhost\r\n\r\n"
	if actual := b.String(); actual != expected {
		t.Fatalf("StartBody: expected %q, got %q", expected, actual)
	}
}

func TestDoubleWriteBody(t *testing.T) {
	var b bytes.Buffer
	c := &writer{Writer: bufio.NewWriter(&b), tmp: &b}
	c.StartBody()
	if err := c.WriteBody(strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
	err := c.WriteBody(strings.NewReader(""))
	expected := `phase error: expected body, got requestline`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}

func TestDoubleWriteChunked(t *testing.T) {
	var b bytes.Buffer
	c := &writer{Writer: bufio.NewWriter(&b), tmp: &b}
	c.StartBody()
	if err := c.WriteChunked(strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
	err := c.WriteChunked(strings.NewReader(""))
	expected := `phase error: expected body, got requestline`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}

type writeTest struct {
	headers  []Header
	body     string
	expected string
}

var writeTests = []writeTest{
	{[]Header{{"Host", "localhost"}, {"Connection", "close"}},
		"abcd1234",
		"Host: localhost\r\nConnection: close\r\n\r\nabcd1234",
	},
}

// test only method, real call will come from Client.
func (c *writer) write(t *testing.T, w writeTest) {
	c.StartHeaders()
	for _, h := range w.headers {
		if err := c.WriteHeader(h.Key, h.Value); err != nil {
			t.Fatal(err)
		}
	}
	c.StartBody()
	if err := c.WriteBody(b(w.body)); err != nil {
		t.Fatal(err)
	}
}

func TestWrite(t *testing.T) {
	for _, tt := range writeTests {
		var b bytes.Buffer
		c := &writer{Writer: bufio.NewWriter(&b), tmp: &b}
		c.write(t, tt)
		if actual := b.String(); actual != tt.expected {
			t.Errorf("TestWrite: expected %q, got %q", tt.expected, actual)
		}
	}
}

var writeBodyTests = []struct {
	io.Reader
	expected string
}{
	{strings.NewReader(""), ""},
	{strings.NewReader("hello world"), "hello world"},
}

func TestWriteBody(t *testing.T) {
	for _, tt := range writeBodyTests {
		var b bytes.Buffer
		w := &writer{Writer: &b, phase: body}
		if err := w.WriteBody(tt.Reader); err != nil {
			t.Fatal(err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("WriteBody: expected %q, got %q", tt.expected, actual)
		}
	}
}

var writeChunkedTests = []struct {
	io.Reader
	expected string
}{
	{strings.NewReader(""), "0\r\n"},
	{strings.NewReader("all your base are belong to us"), "1e\r\nall your base are belong to us\r\n0\r\n"},
}

func TestWriteChunked(t *testing.T) {
	for _, tt := range writeChunkedTests {
		var b bytes.Buffer
		w := &writer{Writer: &b, phase: body}
		if err := w.WriteChunked(tt.Reader); err != nil {
			t.Fatal(err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("WriteBody: expected %q, got %q", tt.expected, actual)
		}
	}
}

var headerBufferingTests = []struct {
	f func(*writer) error
	n int
}{
	{
		func(w *writer) error {
			return w.WriteRequestLine("GET", "/", nil, HTTP_1_1.String())
		},
		0,
	},
	{
		func(w *writer) error {
			return w.WriteRequestLine("GET", "/foo", []string{"bar", "baz"}, HTTP_1_1.String())
		},
		0,
	},
	{
		func(w *writer) error {
			if err := w.WriteRequestLine("GET", "/foo", []string{"bar", "baz"}, HTTP_1_1.String()); err != nil {
				return err
			}
			return w.WriteHeader("Host", "localhost")
		},
		0,
	},
	{
		func(w *writer) error {
			if err := w.WriteRequestLine("GET", "/", nil, HTTP_1_1.String()); err != nil {
				return err
			}
			return w.StartBody()
		},
		1,
	},
	{
		func(w *writer) error {
			if err := w.WriteRequestLine("GET", "/foo", []string{"bar", "baz"}, HTTP_1_1.String()); err != nil {
				return err
			}
			return w.StartBody()
		},
		1,
	},
	{
		func(w *writer) error {
			if err := w.WriteRequestLine("GET", "/foo", []string{"bar", "baz"}, HTTP_1_1.String()); err != nil {
				return err
			}
			return w.WriteHeader("Host", "localhost")
		},
		0,
	},
	{
		func(w *writer) error {
			if err := w.WriteRequestLine("GET", "/foo", []string{"bar", "baz"}, HTTP_1_1.String()); err != nil {
				return err
			}
			if err := w.WriteHeader("Host", "localhost"); err != nil {
				return err
			}
			return w.StartBody()
		},
		1,
	},
	{
		func(w *writer) error {
			if err := w.WriteRequestLine("GET", "/foo", []string{"bar", "baz"}, HTTP_1_1.String()); err != nil {
				return err
			}
			for _, h := range []Header{{"Host", "localhost"}, {"Connection", "close"}} {
				if err := w.WriteHeader(h.Key, h.Value); err != nil {
					return err
				}
			}
			return w.StartBody()
		},
		1,
	},
	{
		func(w *writer) error {
			if err := w.WriteRequestLine("GET", "/foo", []string{"bar", "baz"}, HTTP_1_1.String()); err != nil {
				return err
			}
			for _, h := range []Header{{"Host", "localhost"}, {"Connection", "close"}} {
				if err := w.WriteHeader(h.Key, h.Value); err != nil {
					return err
				}
			}
			if err := w.StartBody(); err != nil {
				return err
			}
			return w.WriteBody(strings.NewReader("Hello world!"))
		},
		2,
	},
}

type countingWriter struct {
	io.Writer
	n int
}

func (w *countingWriter) Write(buf []byte) (int, error) {
	w.n++
	return w.Writer.Write(buf)
}

// verify that header buffering works
func TestHeaderBuffering(t *testing.T) {
	for _, tt := range headerBufferingTests {
		cw := countingWriter{Writer: ioutil.Discard}
		w := &writer{Writer: &cw}
		if err := tt.f(w); err != nil {
			t.Fatal(err)
		}
		if cw.n != tt.n {
			t.Errorf("expected %d writes, got %d", tt.n, cw.n)
		}
	}
}
