package client

import (
	"bufio"
	"bytes"
	"errors"
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
	reader *bufio.Reader
}

const readerBuffer = 4096

// newConn returns a new *Conn
func newConn(rw io.ReadWriter) *Conn {
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

// ReadVersion reads a HTTP version string from the wire.
func (c *Conn) ReadVersion() (Version, error) {
	var major, minor int
	for pos := 0; pos < len("HTTP/x.x "); pos++ {
		c, err := c.reader.ReadByte()
		if err != nil {
			return invalidVersion, err
		}
		switch pos {
		case 0:
			if c != 'H' {
				return readVersionErr(pos, 'H', c)
			}
		case 1, 2:
			if c != 'T' {
				return readVersionErr(pos, 'T', c)
			}
		case 3:
			if c != 'P' {
				return readVersionErr(pos, 'P', c)
			}
		case 4:
			if c != '/' {
				return readVersionErr(pos, '/', c)
			}
		case 5:
			switch c {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				major = int(int(c) - 0x30)
			}
		case 6:
			if c != '.' {
				return readVersionErr(pos, '.', c)
			}
		case 7:
			switch c {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				minor = int(int(c) - 0x30)
			}
		case 8:
			if c != ' ' {
				return readVersionErr(pos, ' ', c)
			}
		}
	}
	return Version{major, minor}, nil
}

var invalidVersion Version

func readVersionErr(pos int, expected, got byte) (Version, error) {
	return invalidVersion, fmt.Errorf("ReadVersion: expected %q, got %q at position %v", expected, got, pos)
}

// ReadStatusCode reads the HTTP status code from the wire.
func (c *Conn) ReadStatusCode() (int, error) {
	var code int
	for pos := 0; pos < len("200 "); pos++ {
		c, err := c.reader.ReadByte()
		if err != nil {
			return 0, err
		}
		switch pos {
		case 0, 1, 2:
			switch c {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				switch pos {
				case 0:
					code = int(int(c)-0x30) * 100
				case 1:
					code += int(int(c)-0x30) * 10
				case 2:
					code += int(int(c) - 0x30)
				}
			}
		case 3:
			if c != ' ' {
				return 0, fmt.Errorf("ReadStatusCode: expected %q, got %q at position %v", ' ', c, pos)
			}
		}
	}
	return code, nil
}

// ReadStatusLine reads the status line.
func (c *Conn) ReadStatusLine() (string, int, string, error) {
	version, err := c.ReadVersion()
	if err != nil {
		return "", 0, "", err
	}
	code, err := c.ReadStatusCode()
	if err != nil {
		return "", 0, "", err
	}
	msg, _, err := c.reader.ReadLine()
	return version.String(), code, string(msg), err
}

// ReadHeader reads a http header.
func (c *Conn) ReadHeader() (string, string, bool, error) {
	line, err := c.readLine()
	if err != nil {
		return "", "", false, err
	}
	if string(line) == "\r\n" {
		return "", "", true, nil
	}
	v := bytes.SplitN(line, []byte(":"), 2)
	if len(v) != 2 {
		return "", "", false, errors.New("invalid header line")
	}
	return string(bytes.TrimSpace(v[0])), string(bytes.TrimSpace(v[1])), false, nil
}

func (c *Conn) ReadBody() io.Reader {
	return c.reader
}

// readLine returns a []byte terminated by a \r\n.
func (c *Conn) readLine() ([]byte, error) {
	return c.reader.ReadBytes('\n')
}
