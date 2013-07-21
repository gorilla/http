package client

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
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
	reader *bufio.Reader
}

const readerBuffer = 4096

// NewConn returns a new *Conn
func NewConn(rw io.ReadWriter) *Conn {
	return &Conn{
		reader: bufio.NewReaderSize(rw, readerBuffer),
		writer: rw,
	}
}

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
func (c *Conn) StartBody() error {
	c.phase = body
	_, err := c.writer.Write([]byte("\r\n"))
	return err
}

// Write body writer the buffer on the wire.
func (c *Conn) WriteBody(r io.Reader) error {
	if c.phase != body {
		return &phaseError{body, c.phase}
	}
	_, err := io.Copy(c.writer, r)
	c.phase = requestline
	return err
}

// ReadStatusLine reads the status line.
func (c *Conn) ReadStatusLine() (int, string, error) {
	line, err := c.readLine()
	if err != nil {
		return 0, "", err
	}
	reader := bytes.NewReader(line)
	var code int
	if _, err := fmt.Fscanf(reader, "%d ", &code); err != nil {
		return 0, "", err
	}
	s := bufio.NewScanner(reader)
	s.Scan()
	return code, s.Text(), s.Err()
}

// ReadHeader reads a http header.
func (c *Conn) ReadHeader() (string, string, bool, error) {
	s := bufio.NewScanner(c.reader)
	if !s.Scan() {
		return "", "", true, s.Err()
	}
	line := s.Text()
	if line == "" {
		return "", "", true, nil // last header line
	}
	v := strings.SplitN(line, ":", 2)
	if len(v) != 2 {
		return "", "", false, errors.New("invalid header line")
	}
	return strings.TrimSpace(v[0]), strings.TrimSpace(v[1]), false, nil
}

// ReadBody returns length bytes of body
// TODO(dfc) temporary function, read readbody will return an io.Reader
// TODO(dfc) doesn't handle chunked transfer encoding
func (c *Conn) ReadBody(length int) ([]byte, error) {
	v := make([]byte, length)
	_, err := io.ReadFull(c.reader, v)
	return v, err
}

// readLine returns a []byte terminated by a \r\n.
func (c *Conn) readLine() ([]byte, error) {
	return c.reader.ReadBytes('\n')
}
