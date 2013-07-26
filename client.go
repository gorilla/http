package http

import (
	"net"
	"net/url"
	"strings"

	"github.com/gorilla/http/client"
)

// Client implements a high level HTTP client. Client methods can be called concurrently
// to as many end points as required.
// Concurrency, connection reuse, caching, and keepalive behavior is managed by the
// ConnectionManager.
type Client struct {
}

type Headers struct{}

func (c *Client) Get(u string) error {
	url, err := url.Parse(u)
	if err != nil {
		return err
	}
	host := url.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}
	var req client.Request
	c1 := client.NewClient(conn)
	if err := c1.SendRequest(&req); err != nil {
		return err
	}
	_, err = c1.ReadResponse()
	return err
}

type rc struct{}

func (r *rc) Read([]byte) (int, error) { return 0, nil }
func (r *rc) Close() error             { return nil }
