// Package gorilla/http is a high level HTTP client.
//
// This package provides high level convience methods for common http operations.
// Additionally a high level HTTP client implementation.
//
// These high level functions are expected to change. Your feedback on their form
// and utility is warmly requested.
//
// Please raise issues at https://github.com/gorilla/http/issues.
//
// For lower level http implementations, see gorilla/http/client.
package http

import (
	"io"
)

// DefaultClient is the default http Client used by this package.
// It's defaults are expected to represent the best practice
// at the time, but may change over time. If you need more
// control or reproducibility, you should construct your own client.
var DefaultClient = Client{
	dialer:          new(dialer),
	FollowRedirects: true,
}

// Get issues a GET request using the DefaultClient and writes the result to
// to w if successful. If the status code of the response is not a success (see
// Success.IsSuccess()) no data will be written and the status code will be
// returned as an error.
func Get(w io.Writer, url string) (int64, error) {
	status, _, r, err := DefaultClient.Get(url, nil)
	if err != nil {
		return 0, err
	}
	defer r.Close()
	if !status.IsSuccess() {
		return 0, &StatusError{status}
	}
	return io.Copy(w, r)
}

// Post issues a POST request using the DefaultClient using r as the body.
// If the status code was not a success code, it will be returned as an error.
func Post(url string, r io.Reader) error {
	status, _, rc, err := DefaultClient.Post(url, nil, r)
	if err != nil {
		return err
	}
	defer rc.Close()
	if !status.IsSuccess() {
		return &StatusError{status}
	}
	return nil
}
