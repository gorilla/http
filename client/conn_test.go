package client

import (
	"bytes"
	"io"
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
	var c Conn
	err := c.WriteHeader("Host", "localhost")
	if _, ok := err.(*phaseError); !ok {
		t.Fatalf("expected %T, got %v", new(phaseError), err)
	}
	expected := `phase error: expected headers, got requestline`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}

func TestnewConn(t *testing.T) {
	var b bytes.Buffer
	newConn(&b)
}

var writeRequestLineTests = []struct {
	method, uri, version string
	expected             string
}{
	{"GET", "/foo", "HTTP/1.0", "GET /foo HTTP/1.0\r\n"},
}

func TestConnWriteRequestLine(t *testing.T) {
	for _, tt := range writeRequestLineTests {
		var b bytes.Buffer
		c := newConn(&b)
		if err := c.WriteRequestLine(tt.method, tt.uri, tt.version); err != nil {
			t.Fatalf("Conn.WriteRequestLine(%q, %q, %q): %v", tt.method, tt.uri, tt.version, err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("Conn.WriteRequestLine(%q, %q, %q): expected %q, got %q", tt.method, tt.uri, tt.version, tt.expected, actual)
		}
	}
}

func TestConnDoubleRequestLine(t *testing.T) {
	var b bytes.Buffer
	c := newConn(&b)
	if err := c.WriteRequestLine("GET", "/hello", "HTTP/0.9"); err != nil {
		t.Fatal(err)
	}
	err := c.WriteRequestLine("GET", "/hello", "HTTP/0.9")
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

func TestConnWriteHeader(t *testing.T) {
	for _, tt := range writeHeaderTests {
		var b bytes.Buffer
		c := newConn(&b)
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
	c := newConn(&b)
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
	c := newConn(new(bytes.Buffer))
	c.StartBody()
	if err := c.WriteBody(b("")); err != nil {
		t.Fatal(err)
	}
	err := c.WriteBody(b(""))
	expected := `phase error: expected body, got requestline`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}

type header struct{ key, value string }
type writeTest struct {
	headers  []header
	body     string
	expected string
}

var writeTests = []writeTest{
	{[]header{{"Host", "localhost"}, {"Connection", "close"}},
		"abcd1234",
		"Host: localhost\r\nConnection: close\r\n\r\nabcd1234",
	},
}

// test only method, real call will come from Client.
func (c *Conn) Write(t *testing.T, w writeTest) {
	c.StartHeaders()
	for _, h := range w.headers {
		if err := c.WriteHeader(h.key, h.value); err != nil {
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
		c := newConn(&b)
		c.Write(t, tt)
		if actual := b.String(); actual != tt.expected {
			t.Errorf("TestWrite: expected %q, got %q", tt.expected, actual)
		}
	}
}

var readVersionTests = []struct {
	line     string
	expected Version
	err      error
}{
	{"HTTP/1.0 ", HTTP_1_0, nil},
	{"HTTP/1.0", HTTP_1_0, nil},
}

func TestReadVersion(t *testing.T) {
	for _, tt := range readVersionTests {
		c := &Conn{reader: b(tt.line)}
		actual, err := c.ReadVersion()
		if actual != tt.expected || err != tt.err {
			t.Errorf("ReadVersion(%q): expected %v %v, got %v %v", tt.line, tt.expected, tt.err, actual, err)
		}
	}
}

var readStatusLineTests = []struct {
	line    string
	version string
	code    int
	msg     string
	err     error
}{
	{"HTTP/1.0 200 OK\r\n", "HTTP/1.0", 200, "OK", nil},
	{"HTTP/1.1 200 OK\r\n\r\n", "HTTP/1.1", 200, "OK", nil},
	{"HTTP/1.1 200 OK", "", 0, "", io.EOF},
	{"HTTP/1.0 200", "", 0, "", io.EOF},
	{"HTTP/1.0", "", 0, "", io.EOF},
}

func TestReadStatusLine(t *testing.T) {
	for _, tt := range readStatusLineTests {
		c := &Conn{reader: b(tt.line)}
		version, code, msg, err := c.ReadStatusLine()
		if version != tt.version || code != tt.code || msg != tt.msg || err != tt.err {
			t.Errorf("ReadStatusLine(%q): expected %q %d %q %v, got %q %d %q %v", tt.line, tt.version, tt.code, tt.msg, tt.err, version, code, msg, err)
		}
	}
}

var readHeaderTests = []struct {
	header     string
	key, value string
	done       bool
}{
	{"Host: localhost\r\n", "Host", "localhost", false},
	{"Host: localhost\r\n\r\n", "Host", "localhost", false},
	{"Connection:close\r\n", "Connection", "close", false},
	{"Connection:close\r\n\r\n", "Connection", "close", false},
	{"Vary : gzip\r\n", "Vary", "gzip", false},
	{"\r\n", "", "", true},
}

func TestReadHeader(t *testing.T) {
	for _, tt := range readHeaderTests {
		c := &Conn{reader: b(tt.header)}
		key, value, done, err := c.ReadHeader()
		if err != nil {
			t.Fatalf("ReadHeader(%q): %v", tt.header, err)
		}
		if key != tt.key || value != tt.value || done != tt.done {
			t.Errorf("ReadHeader: expected %q %q %v, got %q %q %v", tt.key, tt.value, tt.done, key, value, done)
		}
	}
}

var readHeadersTests = []struct {
	headers  string
	expected []Header
	done     bool
}{
	{"Host: localhost\r\n", []Header{{"Host", "localhost"}}, false},
	{"Host: localhost\r\n\r\n", []Header{{"Host", "localhost"}}, true},
	{"Connection:close\r\n", []Header{{"Connection", "close"}}, false},
	{"Connection:close\r\n\r\n", []Header{{"Connection", "close"}}, true},
	{"Vary : gzip\r\n", []Header{{"Vary", "gzip"}}, false},
	{"\r\n", nil, true},
	{"Host: localhost\r\nConnection:close\r\n", []Header{{"Host", "localhost"}, {"Connection", "close"}}, false},
	{"Host: localhost\r\nConnection:close\r\n\r\n", []Header{{"Host", "localhost"}, {"Connection", "close"}}, true},
}

func TestReadHeaders(t *testing.T) {
NEXT:
	for _, tt := range readHeadersTests {
		c := &Conn{reader: b(tt.headers)}
		for i, done := 0, false; !done; i++ {
			var key, value string
			var err error
			key, value, done, err = c.ReadHeader()
			if err == io.EOF {
				break NEXT
			}
			if err != nil {
				t.Errorf("ReadHeader(%q): %v", tt.headers, err)
				break NEXT
			}
			h := tt.expected[i]
			if key != h.Key || value != h.Value {
				t.Errorf("ReadHeader(%q): expected %q %q, got %q %q", tt.headers, h.Key, h.Value, key, value)
				break NEXT
			}
		}
	}
}

var readBodyTests = []struct {
	body     string
	length   int
	expected string
	err      error
}{
	{"hello", len("hello"), "hello", nil},
	{"hello", len("hello") - 1, "hell", nil},
	{"hello", len("hello") + 1, "hello\x00", io.ErrUnexpectedEOF}, // tests internal behavior
}

// disabled til I know what ReadBody should look like
func testReadBody(t *testing.T) {
	for _, tt := range readBodyTests {
		c := &Conn{reader: b(tt.body)}
		r := c.ReadBody()
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if actual := buf.String(); actual != tt.expected || err != tt.err {
			t.Errorf("ReadBody(%q): expected %q %v , got %q %v", tt.body, tt.expected, tt.err, actual, err)
		}
	}
}

var readLineTests = []struct {
	line     string
	expected string
	err      error
}{
	{"200 OK\r\n", "200 OK\r\n", nil},
	{"200 OK\n", "200 OK\n", nil},
	{"200 OK\r\n\r\n", "200 OK\r\n", nil},
	{"200 OK", "200 OK", io.EOF},
	{"200 ", "200 ", io.EOF},
	{"200", "200", io.EOF},
}

func TestReadLine(t *testing.T) {
	for _, tt := range readLineTests {
		c := &Conn{reader: b(tt.line)}
		actual, err := c.readLine()
		if actual := string(actual); actual != tt.expected || err != tt.err {
			t.Errorf("readLine(%q): expected %q %v, got %q, %v", tt.line, tt.expected, tt.err, actual, err)
		}
	}
}
