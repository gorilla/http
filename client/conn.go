package client

import (
	"fmt"
	"io"
)

type phase int

const (
	requestline phase = iota
	headers
	body
)

func (p phase) String() string {
	switch p {
	case requestline:
		return "requestline"
	case headers:
		return "headers"
	case body:
		return "body"
	default:
		return "UNKNOWN"
	}
}

type phaseError struct {
	expected, got phase
}

func (p *phaseError) Error() string {
	return fmt.Sprintf("phase error: expected %s, got %s", p.expected, p.got)
}

// Conn represents the lowest layer of the HTTP client. It is concerned purely with encoding
// and decoding HTTP messages on the wire.
type Conn struct {
	phase
	writer io.Writer
}

// NewConn returns a new *Conn
func NewConn(w io.Writer) *Conn { return &Conn{writer: w} }

// StartHeaders moves the Conn into the headers phase
func (c *Conn) StartHeaders() { c.phase = headers }

// WriteRequestLine writes the RequestLine and moves the Conn to the headers phase
func (c *Conn) WriteRequestLine(method, uri, version string) error {
	if c.phase != requestline {
		return &phaseError{requestline, c.phase}
	}
	_, err := fmt.Fprintf(c.writer, "%s %s %s\r\n", method, uri, version)
	c.StartHeaders()
	return err
}

// WriteHeader writes the canonical header form to the wire.
func (c *Conn) WriteHeader(key, value string) error {
	if c.phase != headers {
		return &phaseError{headers, c.phase}
	}
	_, err := fmt.Fprintf(c.writer, "%s: %s\r\n", key, value)
	return err
}

// StartBody moves the Conn into the body phase, no further headers may be sent at this point.
func (c *Conn) StartBody() {
	c.phase = body
	c.writer.Write([]byte("\r\n")) // ignore error, the call to WriteBody will expose it.
}

// Write body writer the buffer on the wire.
func (c *Conn) WriteBody(buf []byte) error {
	if c.phase != body {
		return &phaseError{body, c.phase}
	}
	_, err := c.writer.Write(buf)
	c.phase = requestline
	return err
}
