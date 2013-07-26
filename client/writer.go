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

type writer struct {
	phase
	io.Writer
}

// StartHeaders moves the Conn into the headers phase
func (w *writer) StartHeaders() { w.phase = headers }

// WriteRequestLine writes the RequestLine and moves the Conn to the headers phase
func (w *writer) WriteRequestLine(method, uri, version string) error {
	if w.phase != requestline {
		return &phaseError{requestline, w.phase}
	}
	_, err := fmt.Fprintf(w, "%s %s %s\r\n", method, uri, version)
	w.StartHeaders()
	return err
}

// WriteHeader writes the canonical header form to the wire.
func (w *writer) WriteHeader(key, value string) error {
	if w.phase != headers {
		return &phaseError{headers, w.phase}
	}
	_, err := fmt.Fprintf(w, "%s: %s\r\n", key, value)
	return err
}

// StartBody moves the Conn into the body phase, no further headers may be sent at this point.
func (w *writer) StartBody() error {
	w.phase = body
	_, err := w.Write([]byte("\r\n"))
	return err
}

// Write body writer the buffer on the wire.
func (w *writer) WriteBody(r io.Reader) error {
	if w.phase != body {
		return &phaseError{body, w.phase}
	}
	_, err := io.Copy(w, r)
	w.phase = requestline
	return err
}
