// Package gorilla/http is a high level HTTP client.
//
// This package provides high level convience methods for common http operations.
// Additionally a high level HTTP client implementation.
//
// For lower level http implementations, see gorilla/http/client.
package http

// Default is the default http Client used by this package.
// It's defaults are expected to represent the best practice
// at the time, but may change over time. If you need more
// control or reliablity, you should construct your own client.
var Default = &Client{}
