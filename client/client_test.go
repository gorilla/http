package client

import (
	"bytes"
	"io"
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
