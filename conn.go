package http

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/gorilla/http/client"
)

// Dialer can dial a remote HTTP server.
type Dialer interface {
	// Dial dials a remote http server returning a Conn having been requested
	// using the given http scheme
	Dial(scheme, host string) (Conn, error)
}

// Conn represnts a connection which can be used to communicate
// with a remote HTTP server.
type Conn interface {
	client.Client
	io.Closer

	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error

	// Release returns the Conn to the Dialer for reuse.
	Release()
}

type conn struct {
	client.Client
	net.Conn
}

func (c *conn) Release() {}

type dialer struct {
	config tls.Config
}

// DefaultDialer is a non-caching dialer for a Client. It is strict with HTTPS
// certificates
var DefaultDialer = dialer{}

// InsecureDialer is a non-caching dialer for a Client. It does not verify
// peer certificates nor does it validate hostnames.
var InsecureDialer = dialer{config: tls.Config{InsecureSkipVerify: true}}

func (d dialer) Dial(scheme, host string) (Conn, error) {
	scheme = strings.ToLower(scheme)
	switch scheme {
	case "http":
		if !strings.Contains(host, ":") {
			host += ":80"
		}
		c, err := net.Dial("tcp", host)
		if err != nil {
			return nil, err
		}
		return &conn{
			Client: client.NewClient(c),
			Conn:   c}, nil
	case "https":
		if !strings.Contains(host, ":") {
			host += ":443"
		}
		c, err := tls.Dial("tcp", host, &d.config)
		if err != nil {
			return nil, err
		}

		return &conn{
			Client: client.NewClient(c),
			Conn:   c}, nil
	default:
		return nil, errors.New(fmt.Sprintf("unsupported scheme: %s", scheme))
	}
}
