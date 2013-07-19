package client

import (
	"fmt"
	"io"
)

type phase int

const (
	headers phase = 1
	body    phase = 2
)

func (p phase) String() string {
	switch p {
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

// StartHeaders moves the Conn into the headers phase
func (c *Conn) StartHeaders() { c.phase = headers }

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
		return &phaseError{headers, c.phase}
	}
	_, err := c.writer.Write(buf)
	return err
}
