// Package gorilla/http/client contains the lower level HTTP client implementation.
//
// Package client is divided into two layers. The upper layer is the client.Client layer
// which handles the vaguaries of http implementations. The lower layer, two types, reader
// and writer are concerned with encoding and decoding the HTTP data stream on the wire.
//
// The interface presented by client.Client is very powerful. It is not expected that normal
// consumers of HTTP services will need to operate at this level and should instead user the
// higher level interfaces in the gorilla/http package.
package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http/httputil"
	"strconv"
	"strings"
)

// Version represents a HTTP version.
type Version struct {
	major, minor int
}

func (v *Version) String() string { return fmt.Sprintf("HTTP/%d.%d", v.major, v.minor) }

var (
	HTTP_1_0 = Version{1, 0}
	HTTP_1_1 = Version{1, 1}
)

// Header represents a HTTP header.
type Header struct {
	Key   string
	Value string
}

// Request represents a complete HTTP request.
type Request struct {
	Method string
	URI    string
	Version

	Headers []Header

	Body io.Reader
}

const readerBuffer = 4096

// Client represents a single connection to a http server. Client obeys KeepAlive conditions for
// HTTP but connection pooling is expected to be handled at a higher layer.
type Client interface {
	WriteRequest(*Request) error
	ReadResponse() (*Response, error)
}

// NewClient returns a Client implementation which uses rw to communicate.
func NewClient(rw io.ReadWriter) Client {
	return &client{
		reader: reader{bufio.NewReaderSize(rw, readerBuffer)},
		writer: writer{Writer: rw},
	}
}

type client struct {
	reader
	writer
}

// SendRequest marshalls a HTTP request to the wire.
func (c *client) WriteRequest(req *Request) error {
	if err := c.WriteRequestLine(req.Method, req.URI, req.Version.String()); err != nil {
		return err
	}
	for _, h := range req.Headers {
		if err := c.WriteHeader(h.Key, h.Value); err != nil {
			return err
		}
	}
	if err := c.StartBody(); err != nil {
		return err
	}
	if req.Body != nil {
		if err := c.WriteBody(req.Body); err != nil {
			return err
		}
	}
	return nil
}

// Status represents an HTTP status code.
type Status struct {
	Code   int
	Reason string
}

func (s Status) String() string { return fmt.Sprintf("%d %s", s.Code, s.Reason) }

var invalidStatus Status

// ReadResponse unmarshalls a HTTP response.
func (c *client) ReadResponse() (*Response, error) {
	version, code, msg, err := c.ReadStatusLine()
	var headers []Header
	if err != nil {
		return nil, fmt.Errorf("ReadStatusLine: %v", err)
	}
	for {
		var key, value string
		var done bool
		key, value, done, err = c.ReadHeader()
		if err != nil || done {
			break
		}
		if key == "" || value == "" {
			err = errors.New("invalid header")
			break
		}
		headers = append(headers, Header{key, value})
	}
	var resp = Response{
		Version: version,
		Status:  Status{code, msg},
		Headers: headers,
		Body:    c.ReadBody(),
	}
	if l := resp.ContentLength(); l >= 0 {
		resp.Body = io.LimitReader(resp.Body, l)
	} else if resp.TransferEncoding() == "chunked" {
		resp.Body = httputil.NewChunkedReader(resp.Body)
	}
	return &resp, err
}

type RequestLine struct {
	Method string
	Path   string
	Version
}

func (r *RequestLine) String() string {
	return fmt.Sprintf("%s %s %s", r.Method, r.Path, r.Version.String())
}

// Response represents an RFC2616 response.
type Response struct {
	Version
	Status
	Headers []Header
	Body    io.Reader
}

// ContentLength returns the length of the body. If the body length is not known
// ContentLength will return -1.
func (r *Response) ContentLength() int64 {
	for _, h := range r.Headers {
		if strings.EqualFold(h.Key, "Content-Length") {
			length, err := strconv.ParseInt(h.Value, 10, 64)
			if err != nil {
				continue
			}
			return int64(length)
		}
	}
	return -1
}

// CloseRequested returns if Reason includes a Connection: close header.
func (r *Response) CloseRequested() bool {
	for _, h := range r.Headers {
		if strings.EqualFold(h.Key, "Connection") {
			return h.Value == "close"
		}
	}
	return false
}

// TransferEncoding returns the transfer encoding this message was transmitted with.
// If not is specified by the sender, "identity" is assumed.
func (r *Response) TransferEncoding() string {
	for _, h := range r.Headers {
		if strings.EqualFold(h.Key, "Transfer-Encoding") {
			switch h.Value {
			case "identity", "chunked":
				return h.Value
			}
		}
	}
	return "identity"
}

// Message represents common traits of both Requests and Responses.
type Message interface {
	ContentLength() int64
	CloseRequested() bool
}
