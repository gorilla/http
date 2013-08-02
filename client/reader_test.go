package client

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

var readVersionTests = []struct {
	line     string
	expected Version
	err      error
}{
	{"HTTP/1.0 ", HTTP_1_0, nil},
	{"HTTP/1.0", Version{}, io.EOF},
	{"http/1.1", Version{}, errors.New("ReadVersion: expected 'H', got 'h' at position 0")},
	{"Http/1.1", Version{}, errors.New("ReadVersion: expected 'T', got 't' at position 1")},
	{"HTTp/1.1", Version{}, errors.New("ReadVersion: expected 'P', got 'p' at position 3")},
	{"HTTP#1.1", Version{}, errors.New("ReadVersion: expected '/', got '#' at position 4")},
	{"HTTP/11", Version{}, errors.New("ReadVersion: expected '.', got '1' at position 6")},
	{"HTTP/1.10", Version{}, errors.New("ReadVersion: expected ' ', got '0' at position 8")},
}

func TestReadVersion(t *testing.T) {
	for _, tt := range readVersionTests {
		c := &reader{b(tt.line)}
		actual, err := c.ReadVersion()
		if actual != tt.expected || !sameErr(err, tt.err) {
			t.Errorf("ReadVersion(%q): expected %v %v, got %v %v", tt.line, tt.expected, tt.err, actual, err)
		}
	}
}

var readStatusCodeTests = []struct {
	line     string
	expected int
	err      error
}{
	{"200 OK\r\n", 200, nil},
	{"200 OK", 200, nil},
	{"200 ", 200, nil},
	{"200", 0, io.EOF},
	{"20 ", 0, io.EOF},
	{"2000", 0, errors.New("ReadStatusCode: expected ' ', got '0' at position 3")},
}

func TestReadStatusCode(t *testing.T) {
	for _, tt := range readStatusCodeTests {
		c := &reader{b(tt.line)}
		actual, err := c.ReadStatusCode()
		if actual != tt.expected || !sameErr(err, tt.err) {
			t.Errorf("ReadVersion(%q): expected %v %v, got %v %v", tt.line, tt.expected, tt.err, actual, err)
		}
	}
}

func sameErr(a, b error) bool {
	if a != nil && b != nil {
		return a.Error() == b.Error()
	}
	return a == b
}

var readStatusLineTests = []struct {
	line string
	Version
	code int
	msg  string
	err  error
}{
	{"HTTP/1.0 200 OK", HTTP_1_0, 200, "OK", nil},
	{"HTTP/1.0 200 OK\r\n", HTTP_1_0, 200, "OK", nil},
	{"HTTP/1.1 200 OK\r\n\r\n", HTTP_1_1, 200, "OK", nil},
	{"HTTP/1.0 200", Version{}, 0, "", io.EOF},
	{"HTTP/1.0 200\r\n", HTTP_1_0, 200, "", nil},
	{"HTTP/1.0", Version{}, 0, "", io.EOF},
	{"", Version{}, 0, "", io.EOF},
}

func TestReadStatusLine(t *testing.T) {
	for _, tt := range readStatusLineTests {
		c := &reader{b(tt.line)}
		version, code, msg, err := c.ReadStatusLine()
		if version != tt.Version || code != tt.code || msg != tt.msg || err != tt.err {
			t.Errorf("ReadStatusLine(%q): expected %q %d %q %v, got %q %d %q %v", tt.line, tt.Version, tt.code, tt.msg, tt.err, version, code, msg, err)
		}
	}
}

var readHeaderTests = []struct {
	header     string
	key, value string
	done       bool
	err        error
}{
	{"Host: localhost\r\n", "Host", "localhost", false, nil},
	{"Host localhost\r\n", "", "", false, errors.New(`invalid header line: "Host localhost\r\n"`)},
	{"Host: localhost", "", "", false, io.EOF},
	{"Host: localhost\r\n\r\n", "Host", "localhost", false, nil},
	{"Connection:close\r\n", "Connection", "close", false, nil},
	{"Connection:close\r\n\r\n", "Connection", "close", false, nil},
	{"Vary : gzip\r\n", "Vary", "gzip", false, nil},
	{"\r\n", "", "", true, nil},
	{"Host: foo\n", "Host", "foo", false, nil},
	{"Pragma: \r\n", "Pragma", "", false, nil},
	// mangled response spotted in the wild
	{"HTTP/1.0 200 OK\r\n", "", "", false, errors.New(`invalid header line: "HTTP/1.0 200 OK\r\n"`)},
}

func TestReadHeader(t *testing.T) {
	for _, tt := range readHeaderTests {
		c := &reader{b(tt.header)}
		key, value, done, err := c.ReadHeader()
		if key != tt.key || value != tt.value || done != tt.done || !sameErr(err, tt.err) {
			t.Errorf("ReadHeader: expected %q %q %v %v, got %q %q %v %v", tt.key, tt.value, tt.done, tt.err, key, value, done, err)
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
	{"Pragma: \r\nContent-Length: 100\r\n\r\n", []Header{{"Pragma", ""}, {"Content-Length", "100"}}, true},
}

func TestReadHeaders(t *testing.T) {
NEXT:
	for _, tt := range readHeadersTests {
		c := &reader{b(tt.headers)}
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
		c := &reader{b(tt.body)}
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
		c := &reader{b(tt.line)}
		actual, err := c.readLine()
		if actual := string(actual); actual != tt.expected || err != tt.err {
			t.Errorf("readLine(%q): expected %q %v, got %q, %v", tt.line, tt.expected, tt.err, actual, err)
		}
	}
}
