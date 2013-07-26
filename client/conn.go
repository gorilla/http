package client

import (
	"bufio"
	"io"
)

// Conn represents the lowest layer of the HTTP client. It is concerned purely with encoding
// and decoding HTTP messages on the wire.
type Conn struct {
	phase
	writer
	reader
}

const readerBuffer = 4096

// newConn returns a new *Conn
func newConn(rw io.ReadWriter) *Conn {
	return &Conn{
		reader: reader{bufio.NewReaderSize(rw, readerBuffer)},
		writer: writer{Writer: rw},
	}
}
