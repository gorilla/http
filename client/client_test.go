package client

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
)

func b(s string) io.Reader { return strings.NewReader(s) }

var sendRequestTests = []struct {
	Request
	expected string
}{
	{Request{
		Method:  "GET",
		URI:     "/",
		Version: "HTTP/1.1",
		// no body
	},
		"GET / HTTP/1.1\r\n\r\n",
	},
	{Request{
		Method:  "GET",
		URI:     "/",
		Version: "HTTP/1.1",
		Body:    b("Hello world!"),
	},
		"GET / HTTP/1.1\r\n\r\nHello world!",
	},
	{Request{
		Method:  "GET",
		URI:     "/",
		Version: "HTTP/1.1",
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
		client := &Client{Conn: NewConn(&b)}
		if err := client.SendRequest(&tt.Request); err != nil {
			t.Fatalf("client.SendRequest(): %v", err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("client.SendRequest(): expected %q, got %q", tt.expected, actual)
		}
	}
}

var readResponseTests = []struct {
	data string
	Status
	headers []Header
	err     error
}{
	{"200 OK\r\n\r\n", Status{200, "OK"}, nil, nil},
	{"404 Not found\r\n\r\n", Status{404, "Not found"}, nil, nil},
}

func TestClientReadResponse(t *testing.T) {
	for _, tt := range readResponseTests {
		client := &Client{Conn: &Conn{reader: b(tt.data)}}
		status, headers, err := client.ReadResponse()
		if status != tt.Status || err != tt.err {
			t.Errorf("client.ReadRequest(): expected %q %v, got %q %v", tt.Status, tt.err, status, err)
		}
		if !reflect.DeepEqual(tt.headers, headers) {
			t.Errorf("client.ReadRequest(): expected %v, got %v", tt.headers, headers)
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
