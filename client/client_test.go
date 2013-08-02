package client

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

// assert that client.client implements client.Client
var _ Client = new(client)

// assert that client.Request and client.Response implement client.Message
// var _ Message = new(Request)
var _ Message = new(Response)

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
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_1,
			// no body
		},
		"GET / HTTP/1.1\r\n\r\n",
	},
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_0,
			// empty body, without len
			Body: b(""),
		},
		"GET / HTTP/1.0\r\n\r\n",
	},
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_1,
			// empty body, without len
			Body: b(""),
		},
		"GET / HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n",
	},
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_0,
			// empty body
			Body: strings.NewReader(""),
		},
		"GET / HTTP/1.0\r\nContent-Length: 0\r\n\r\n",
	},
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_1,
			// empty body
			Body: strings.NewReader(""),
		},
		"GET / HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
	},
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_0,
			Body:    b("Hello world!"),
		},
		"GET / HTTP/1.0\r\n\r\nHello world!",
	},
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_1,
			Body:    b("Hello world!"),
		},
		"GET / HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\nc\r\nHello world!\r\n0\r\n",
	},
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_0,
			Body:    strings.NewReader("Hello world!"),
		},
		"GET / HTTP/1.0\r\nContent-Length: 12\r\n\r\nHello world!",
	},
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_1,
			Body:    strings.NewReader("Hello world!"),
		},
		"GET / HTTP/1.1\r\nContent-Length: 12\r\n\r\nHello world!",
	},
	{
		Request{
			Method:  "GET",
			Path:    "/",
			Version: HTTP_1_1,
			Body:    strings.NewReader("Hello world!"),
			Headers: []Header{{
				Key: "Host", Value: "localhost",
			}},
		},
		"GET / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 12\r\n\r\nHello world!",
	},
	{
		Request{
			Method:  "POST",
			Path:    "/foo",
			Version: HTTP_1_0,
			Body:    strings.NewReader("hello"),
		},
		"POST /foo HTTP/1.0\r\nContent-Length: 5\r\n\r\nhello",
	},
}

func TestClientSendRequest(t *testing.T) {
	for _, tt := range sendRequestTests {
		var b bytes.Buffer
		client := NewClient(&b)
		if err := client.WriteRequest(&tt.Request); err != nil {
			t.Fatalf("client.SendRequest(): %v", err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("client.SendRequest(): expected %q, got %q", tt.expected, actual)
		}
	}
}

var readResponseTests = []struct {
	data string
	*Response
	err error
}{
	{"HTTP/1.0 200 OK\r\n\r\n",
		&Response{
			Version: HTTP_1_0,
			Status:  Status{200, "OK"},
		},
		nil},
	{"HTTP/1.0 200 OK\r\n",
		&Response{
			Version: HTTP_1_0,
			Status:  Status{200, "OK"},
		},
		io.EOF},
	{"HTTP/1.1 404 Not found\r\n\r\n",
		&Response{
			Version: HTTP_1_1,
			Status:  Status{404, "Not found"},
		},
		nil},
	{"HTTP/1.1 404 Not found\r\n",
		&Response{
			Version: HTTP_1_1,
			Status:  Status{404, "Not found"},
		}, io.EOF},
	{"HTTP/1.0 200 OK\r\nHost: localhost\r\n\r\n",
		&Response{
			Version: HTTP_1_0,
			Status:  Status{200, "OK"},
			Headers: []Header{{"Host", "localhost"}},
		}, nil},
	{"HTTP/1.1 200 OK\r\nHost: localhost\r\n",
		&Response{
			Version: HTTP_1_1,
			Status:  Status{200, "OK"},
			Headers: []Header{{"Host", "localhost"}},
		}, io.EOF},
	{"HTTP/1.0 200 OK\r\nHost: localhost\r\nConnection : close\r\n",
		&Response{
			Version: HTTP_1_0,
			Status:  Status{200, "OK"},
			Headers: []Header{{"Host", "localhost"}, {"Connection", "close"}},
		}, io.EOF},
	// excessively long chunk.
	{"HTTP/1.1 200 Ok\r\nTransfer-encoding: chunked\r\n\r\n" +
		"004\r\n1234\r\n" +
		"1000000000000000000001\r\n@\r\n" +
		"00000000\r\n" +
		"\r\n",
		&Response{
			Version: HTTP_1_1,
			Status:  Status{200, "Ok"},
			Headers: []Header{{"Transfer-encoding", "chunked"}},
			Body:    b("1234@"),
		},
		nil,
	},
	// silly redirect header
	{
		"HTTP/1.1 302 WebCenter Redirect\r\n" +
			"Connection: close\r\n" +
			"Date: Mon, 29 Jul 2013 22:18:39 GMT\r\n" +
			"Location: http://kcsawsp01.contactcenter.ktb.co.th:80/\r\n" +
			"HTTP/1.1 400 Bad Request\r\n" +
			"Content-Type: text/html\r\n" +
			"Content-Length: 87\r\n" +
			"Connection: close\r\n" +
			"\r\n" +
			"<html><head><title>Error</title></head><body>The parameter is incorrect. </body></html>\n",
		&Response{
			Version: HTTP_1_1,
			Status:  Status{302, "WebCenter Redirect"},
			Headers: []Header{
				{"Connection", "close"},
				{"Date", "Mon, 29 Jul 2013 22:18:39 GMT"},
				{"Location", "http://kcsawsp01.contactcenter.ktb.co.th:80/"},
			},
			Body: strings.NewReader("Content-Type: text/html\r\nContent-Length: 87\r\nConnection: close\r\n\r\n<html><head><title>Error</title></head><body>The parameter is incorrect. </body></html>\n"),
		},
		errors.New(`invalid header line: "HTTP/1.1 400 Bad Request\r\n"`),
	},
	// totally broken
	{
		"HTTP/1.0 301 Moved Permanently\r\n" +
			"HTTP/1.0 400 Bad Request\r\n",
		&Response{
			Version: HTTP_1_0,
			Status:  Status{301, "Moved Permanently"},
		},
		errors.New(`invalid header line: "HTTP/1.0 400 Bad Request\r\n"`),
	},
	// whitespace status code prefix
	{
		"HTTP/1.0  401 Unauthorized\r\n" +
			"Content-type: text/html\r\n" +
			"WWW-Authenticate: Basic realm=\"NXU-2\"\r\n" +
			"<HTML><BODY><H1>Your Authentication failed<BR></H1><B>Your Request was denied <BR>You do not have permission to view this page</B><BR></BODY></HTML>",
		&Response{
			Version: HTTP_1_0,
			Status:  Status{401, "Unauthorized"},
		},
		errors.New("ReadStatusLine: ReadStatusCode: expected ' ', got '1' at position 3"),
	},
}

func TestClientReadResponse(t *testing.T) {
	for _, tt := range readResponseTests {
		client := &client{reader: reader{b(tt.data)}}
		resp, err := client.ReadResponse()
		if !sameErr(err, tt.err) {
			t.Errorf("client.ReadResponse(%q): expected %q, got %q", tt.data, tt.err, err)
			continue
		}
		if resp == nil {
			continue
		}

		if !reflect.DeepEqual(tt.Response.Headers, resp.Headers) {
			t.Errorf("client.ReadResponse(%q): expected %v, got %v", tt.data, tt.Response.Headers, resp.Headers)
			continue
		}
		if resp.Version != tt.Response.Version || resp.Status != tt.Response.Status {
			t.Errorf("client.ReadResponse(%q): expected %q %q, got %q %q", tt.data, tt.Response.Version, tt.Response.Status, resp.Version, resp.Status)
			continue
		}
		var buf bytes.Buffer
		var expected, actual string
		if tt.Response.Body != nil {
			_, err = io.Copy(&buf, tt.Response.Body)
			expected = buf.String()
		}
		buf.Reset()
		if resp.Body != nil {
			_, err = io.Copy(&buf, resp.Body)
			actual = buf.String()
		}
		if actual != expected {
			t.Errorf("client.ReadResponse(%q): expected %q, got %q", tt.data, expected, actual)
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
	{"HTTP/1.0 200 OK\r\nContent-Length: seven\r\n\r\n", -1},
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

var transferEncodingTests = []struct {
	data     string
	expected string
}{
	{"HTTP/1.0 200 OK\r\n\r\nfoo", "identity"},
	{"HTTP/1.0 200 OK\r\nConnection: close\r\n\r\nfoo", "identity"},
	{"HTTP/1.0 200 OK\r\nConnection: close\r\nTransfer-Encoding: chunked\r\n\r\nfoo", "chunked"},
	{"HTTP/1.1 200 OK\r\n\r\nfoo", "identity"},
	{"HTTP/1.1 200 OK\r\nConnection: close\r\n\r\nfoo", "identity"},
	{"HTTP/1.1 200 OK\r\nConnection: close\r\nTransfer-Encoding: chunked\r\n\r\nfoo", "chunked"},
}

func TestTransferEncoding(t *testing.T) {
	for _, tt := range transferEncodingTests {
		client := &client{reader: reader{b(tt.data)}}
		resp, err := client.ReadResponse()
		if err != nil {
			t.Fatal(err)
		}
		if actual := resp.TransferEncoding(); actual != tt.expected {
			t.Errorf("ReadResponse(%q): TransferEncoding: expected %d got %d", tt.data, tt.expected, actual)
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

var requestContentLengthTests = []struct {
	Request
	expected int64
}{
	{Request{Body: nil}, -1},
	{Request{Body: bytes.NewBuffer([]byte("hello world"))}, 11},
	{Request{Body: strings.NewReader("hello world")}, 11},
}

func TestRequestContentLength(t *testing.T) {
	for _, tt := range requestContentLengthTests {
		actual := tt.Request.ContentLength()
		if tt.expected != actual {
			t.Errorf("Request.ContentLength: expected %d, got %d", tt.expected, actual)
		}
	}
}
