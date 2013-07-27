package http

import (
	"io"
	"net"
	"time"

	"github.com/gorilla/http/client"
)

// Dialer can dial a remote HTTP server.
type Dialer interface {
	// Dial dials a remote http server returning a Conn.
	Dial(network, addr string) (Conn, error)
}

type dialer struct {
}

func (d *dialer) Dial(network, addr string) (Conn, error) {
	c, err := net.Dial(network, addr)
	return &conn{
		Client: client.NewClient(c),
		Conn:   c,
	}, err
}

// Conn represnts a connection which can be used to communicate
// with a remote HTTP server.
type Conn interface {
	client.Client
	io.Closer

	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type conn struct {
	client.Client
	net.Conn
}
