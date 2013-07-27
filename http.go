// Package gorilla/http is a high level HTTP client.
//
// This package provides high level convience methods for common http operations.
// Additionally a high level HTTP client implementation.
//
// For lower level http implementations, see gorilla/http/client.
package http

import (
	"io"
)

// DefaultClient is the default http Client used by this package.
// It's defaults are expected to represent the best practice
// at the time, but may change over time. If you need more
// control or reliablity, you should construct your own client.
var DefaultClient = Client{dialer: &dialer{}}

// Get issues a GET request using the DefaultClient and writes the result to
// to w if successful.
func Get(w io.Writer, url string) (int64, error) {
	_, _, r, err := DefaultClient.Get(url, nil)
	if err != nil {
		return 0, err
	}
	defer r.Close()
	return io.Copy(w, r)
}
