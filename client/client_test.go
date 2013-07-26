package client

import (
	"bufio"
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
)

// assert that client.client implements client.Client
var _ Client = new(client)

func TestNewClient(t *testing.T) {
	var b bytes.Buffer
	var r io.ReadWriter = &b
	var _ Client = NewClient(r)
}

func b(s string) *bufio.Reader { return bufio.NewReader(strings.NewReader(s)) }

var sendRequestTests = []struct {
	Request
	expected string
}{
	{Request{
		Method:  "GET",
		URI:     "/",
		Version: HTTP_1_1,
		// no body
	},
		"GET / HTTP/1.1\r\n\r\n",
	},
	{Request{
		Method:  "GET",
		URI:     "/",
		Version: HTTP_1_1,
		Body:    b("Hello world!"),
	},
		"GET / HTTP/1.1\r\n\r\nHello world!",
	},
	{Request{
		Method:  "GET",
		URI:     "/",
		Version: HTTP_1_1,
		Body:    b("Hello world!"),
		Headers: []Header{{
			Key: "Host", Value: "localhost",
		}},
	},
		"GET / HTTP/1.1\r\nHost: localhost\r\n\r\nHello world!",
	},
}

func TestClientSendRequest(t *testing.T) {
	for _, tt := range sendRequestTests {
		var b bytes.Buffer
		client := NewClient(&b)
		if err := client.SendRequest(&tt.Request); err != nil {
			t.Fatalf("client.SendRequest(): %v", err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("client.SendRequest(): expected %q, got %q", tt.expected, actual)
		}
	}
}

func sl(v Version, code int, msg string) StatusLine { return StatusLine{v, Status{code, msg}} }

var readResponseTests = []struct {
	data string
	*Response
	err error
}{
	{"HTTP/1.0 200 OK\r\n\r\n",
		&Response{
			StatusLine: sl(HTTP_1_0, 200, "OK"),
		},
		nil},
	{"HTTP/1.0 200 OK\r\n",
		&Response{
			StatusLine: sl(HTTP_1_0, 200, "OK"),
		},
		io.EOF},
	{"HTTP/1.1 404 Not found\r\n\r\n",
		&Response{
			StatusLine: sl(HTTP_1_1, 404, "Not found"),
		},
		nil},
	{"HTTP/1.1 404 Not found\r\n",
		&Response{
			StatusLine: sl(HTTP_1_1, 404, "Not found"),
		}, io.EOF},
	{"HTTP/1.0 200 OK\r\nHost: localhost\r\n\r\n",
		&Response{
			StatusLine: sl(HTTP_1_0, 200, "OK"),
			Headers:    []Header{{"Host", "localhost"}},
		}, nil},
	{"HTTP/1.1 200 OK\r\nHost: localhost\r\n",
		&Response{
			StatusLine: sl(HTTP_1_1, 200, "OK"),
			Headers:    []Header{{"Host", "localhost"}},
		}, io.EOF},
	{"HTTP/1.0 200 OK\r\nHost: localhost\r\nConnection : close\r\n",
		&Response{
			StatusLine: sl(HTTP_1_0, 200, "OK"),
			Headers:    []Header{{"Host", "localhost"}, {"Connection", "close"}},
		}, io.EOF},
}

func TestClientReadResponse(t *testing.T) {
	for _, tt := range readResponseTests {
		client := &client{reader: reader{b(tt.data)}}
		resp, err := client.ReadResponse()
		if resp.StatusLine != tt.Response.StatusLine {
			t.Errorf("client.ReadResponse(%q): expected %q, got %q", tt.data, resp.StatusLine, tt.Response.StatusLine)
			continue
		}
		if !reflect.DeepEqual(tt.Response.Headers, resp.Headers) || err != tt.err {
			t.Errorf("client.ReadResponse(%q): expected %v %v, got %v %v", tt.data, tt.Response.Headers, tt.err, resp.Headers, err)
		}
		if err != nil {
			continue
		}
		var buf bytes.Buffer
		var expected, actual string
		if tt.Response.Body != nil {
			_, err = io.Copy(&buf, tt.Response.Body)
			expected = buf.String()
		}
		if resp.Body != nil {
			_, err = io.Copy(&buf, resp.Body)
			actual = buf.String()
		}
		if actual != expected || err != tt.err {
			t.Errorf("client.ReadResponse(%q): expected %q %v, got %q %v", tt.data, expected, tt.err, actual, err)
		}
	}
}

var responseContentLengthTests = []struct {
	data     string
	expected int64
}{
	{"HTTP/1.0 200 OK\r\n\r\n", -1},
	{"HTTP/1.0 200 OK\r\n\r\n ", -1},
	{"HTTP/1.0 200 OK\r\nContent-Length: 1\r\n\r\n ", 1},
	{"HTTP/1.0 200 OK\r\nContent-Length: 0\r\n\r\n", 0},
	{"HTTP/1.0 200 OK\r\nContent-Length: 4294967296\r\n\r\n", 4294967296},
}

func TestResponseContentLength(t *testing.T) {
	for _, tt := range responseContentLengthTests {
		client := &client{reader: reader{b(tt.data)}}
		resp, err := client.ReadResponse()
		if err != nil {
			t.Fatal(err)
		}
		if actual := resp.ContentLength(); actual != tt.expected {
			t.Errorf("ReadResponse(%q): ContentLength: expected %d got %d", tt.data, tt.expected, actual)
		}
	}
}

var closeRequestedTests = []struct {
	data     string
	expected bool
}{
	{"HTTP/1.0 200 OK\r\n\r\nfoo", false},
	{"HTTP/1.0 200 OK\r\nConnection: close\r\n\r\nfoo", true},
	{"HTTP/1.1 200 OK\r\n\r\nfoo", false},
	{"HTTP/1.1 200 OK\r\nConnection: close\r\n\r\nfoo", true},
}

func TestRequestCloseRequested(t *testing.T) {
	for _, tt := range closeRequestedTests {
		client := &client{reader: reader{b(tt.data)}}
		resp, err := client.ReadResponse()
		if err != nil {
			t.Fatal(err)
		}
		if actual := resp.CloseRequested(); actual != tt.expected {
			t.Errorf("ReadResponse(%q): CloseRequested: expected %d got %d", tt.data, tt.expected, actual)
		}
	}
}

var statusStringTests = []struct {
	Status
	expected string
}{
	{Status{200, "OK"}, "200 OK"},
	{Status{418, "I'm a teapot"}, "418 I'm a teapot"},
}

func TestStatusString(t *testing.T) {
	for _, tt := range statusStringTests {
		if actual := tt.Status.String(); actual != tt.expected {
			t.Errorf("Status{%d, %q}.String(): expected %q, got %q", tt.Status.Code, tt.Status.Message, tt.expected, actual)
		}
	}
}

var versionStringTests = []struct {
	Version
	expected string
}{
	{Version{0, 9}, "HTTP/0.9"},
	{Version{1, 0}, "HTTP/1.0"},
	{Version{1, 1}, "HTTP/1.1"},
	{Version{2, 0}, "HTTP/2.0"},
}

func TestVersionString(t *testing.T) {
	for _, tt := range versionStringTests {
		if actual := tt.Version.String(); actual != tt.expected {
			t.Errorf("Version{%d, %d}.String(): expected %q, got %q", tt.Version.major, tt.Version.minor, tt.expected, actual)
		}
	}
}

var requestLineStringTests = []struct {
	RequestLine
	expected string
}{
	{RequestLine{"GET", "/", HTTP_1_0}, "GET / HTTP/1.0"},
	{RequestLine{"PUT", "/foo", HTTP_1_1}, "PUT /foo HTTP/1.1"},
}

func TestRequestLineString(t *testing.T) {
	for _, tt := range requestLineStringTests {
		if actual := tt.RequestLine.String(); actual != tt.expected {
			t.Errorf("RequestLine{%q %q, %q}.String(): expected %q, got %q", tt.RequestLine.Method, tt.RequestLine.Path, tt.RequestLine.Version, tt.expected, actual)
		}
	}
}

var statusLineStringTests = []struct {
	StatusLine
	expected string
}{
	{StatusLine{HTTP_1_0, Status{200, "OK"}}, "HTTP/1.0 200 OK"},
}

func TestStatusLineString(t *testing.T) {
	for _, tt := range statusLineStringTests {
		if actual := tt.StatusLine.String(); actual != tt.expected {
			t.Errorf("StatusLine(%q, %q).String(): expected %q, got %q", tt.StatusLine.Version, tt.StatusLine.Status.String(), tt.expected, actual)
		}
	}
}
