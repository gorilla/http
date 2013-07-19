// Package gorilla/http/client contains the lower level HTTP client implementation.
//
// Package client is divided into two layers. The upper layer is the client.Client layer
// which handles the vaguaries of http implementations. The lower layer, client.Conn is
// concerned with encoding and decoding the HTTP data stream on the wire.
//
// The interface presented by client.Client is very powerful. It is not expected that normal
// consumers of HTTP services will need to operate at this level and should instead user the 
// higher level interfaces in the gorilla/http package.
package client

import (
	"io"
)

// Client represents a single connection to a http server. Client obeys KeepAlive conditions for 
// HTTP but connection pooling is expected to be handled at a higher layer.

// Conn represents the lowest layer of the HTTP client. It is concerned purely with encoding
// and decoding HTTP messages on the wire. Conn over anything that can provide an io.ReadWriteCloser
type Conn struct {
	reader io.Reader
	writer io.Writer
	closer io.Closer
}
