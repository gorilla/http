package http 

import (
	"io"
)

// Client implements a high level HTTP client. Client methods can be called concurrently
// to as many end points as required.
// Concurrency, connection reuse, caching, and keepalive behavior is managed by the
// ConnectionManager.
type Client struct {


}

type Headers struct { }

func (c *Client) Get(url string) (io.ReadCloser, *Headers, error) { return nil, nil, nil }
