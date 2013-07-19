package client

import (
	"bytes"
	"testing"
)

var sendRequestTests = []struct {
	Request
	expected string
}{}

func TestClientSendRequest(t *testing.T) {
	for _, tt := range sendRequestTests {
		var b bytes.Buffer
		client := &Client{Conn: Conn{writer: &b}}
		if err := client.SendRequest(&tt.Request); err != nil {
			t.Fatalf("client.SendRequest(): %v", err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("client.SendRequest(): expected %q, got %q", tt.expected, actual)
		}
	}
}
