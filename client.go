package http

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	stdurl "net/url"
	"strings"

	"github.com/gorilla/http/client"
)

// Client implements a high level HTTP client. Client methods can be called concurrently
// to as many end points as required.
type Client struct {
	dialer Dialer

	// FollowRedirects instructs the client to follow 301/302 redirects when idempotent.
	FollowRedirects bool
}

// Do sends an HTTP request and returns an HTTP response. If the response body is non nil
// it must be closed.
func (c *Client) Do(method, url string, headers map[string][]string, body io.Reader) (client.Status, map[string][]string, io.ReadCloser, error) {
	if headers == nil {
		headers = make(map[string][]string)
	}
	u, err := stdurl.ParseRequestURI(url)
	if err != nil {
		return client.Status{}, nil, nil, err
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	headers["Host"] = []string{host}
	path := u.Path
	if path == "" {
		path = "/"
	}
	conn, err := c.dialer.Dial("tcp", host)
	if err != nil {
		return client.Status{}, nil, nil, err
	}
	req := toRequest(method, path, nil, headers, body)
	if err := conn.WriteRequest(req); err != nil {
		return client.Status{}, nil, nil, err
	}
	resp, err := conn.ReadResponse()
	if err != nil {
		return client.Status{}, nil, nil, err
	}
	rheaders := fromHeaders(resp.Headers)
	rbody := resp.Body
	if h := strings.Join(rheaders["Content-Encoding"], " "); h == "gzip" {
		rbody, err = gzip.NewReader(rbody)
	}
	rc := &readCloser{rbody, conn}
	if resp.Status.IsRedirect() && c.FollowRedirects {
		// consume the response body
		_, err := io.Copy(ioutil.Discard, rc)
		err2 := rc.Close()
		if err != nil || err2 != nil {
			return client.Status{}, nil, nil, err // TODO
		}
		loc := strings.Join(rheaders["Location"], " ")
		if strings.HasPrefix(loc, "/") {
			loc = fmt.Sprintf("http://%s%s", host, loc)
		}
		return c.Do(method, loc, headers, body)
	}
	return resp.Status, rheaders, rc, err
}

// StatusError reprents a client.Status as an error.
type StatusError struct {
	client.Status
}

func (s *StatusError) Error() string {
	return s.Status.String()
}

type readCloser struct {
	io.Reader
	io.Closer
}

// Get sends a GET request. If the response body is non nil it must be closed.
func (c *Client) Get(url string, headers map[string][]string) (client.Status, map[string][]string, io.ReadCloser, error) {
	return c.Do("GET", url, headers, nil)
}

// Post sends a POST request, suppling the contents of the reader as the request body.
func (c *Client) Post(url string, headers map[string][]string, body io.Reader) (client.Status, map[string][]string, io.ReadCloser, error) {
	return c.Do("POST", url, headers, body)
}

func toRequest(method string, path string, query []string, headers map[string][]string, body io.Reader) *client.Request {
	return &client.Request{
		Method:  method,
		Path:    path,
		Query:   query,
		Version: client.HTTP_1_1,
		Headers: toHeaders(headers),
		Body:    body,
	}
}

func toHeaders(h map[string][]string) []client.Header {
	var r []client.Header
	for k, v := range h {
		for _, v := range v {
			r = append(r, client.Header{k, v})
		}
	}
	return r
}

func fromHeaders(h []client.Header) map[string][]string {
	if h == nil {
		return nil
	}
	var r = make(map[string][]string)
	for _, hh := range h {
		r[hh.Key] = append(r[hh.Key], hh.Value)
	}
	return r
}
