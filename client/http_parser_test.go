package client

// These tests are adapted from the node.js/http_parser tests suite
// https://github.com/joyent/node/blob/master/deps/http_parser/test.c
// http_parser is listed as a MIT compatible licence.

import (
	"bytes"
	"testing"
)

var requestTests = []struct {
	name string
	Request
	expected string
	err      error
}{
	{
		name: "curl get",
		Request: Request{
			Method:  "GET",
			Path:    "/test",
			Version: HTTP_1_1,
			Headers: []Header{
				{"User-Agent", "curl/7.18.0 (i486-pc-linux-gnu) libcurl/7.18.0 OpenSSL/0.9.8g zlib/1.2.3.3 libidn/1.1"},
				{"Host", "0.0.0.0=5000"},
				{"Accept", "*/*"},
			},
		},
		expected: "GET /test HTTP/1.1\r\n" +
			"User-Agent: curl/7.18.0 (i486-pc-linux-gnu) libcurl/7.18.0 OpenSSL/0.9.8g zlib/1.2.3.3 libidn/1.1\r\n" +
			"Host: 0.0.0.0=5000\r\n" +
			"Accept: */*\r\n" +
			"\r\n",
	},
	{
		name: "firefox get",
		Request: Request{
			Method:  "GET",
			Path:    "/favicon.ico",
			Version: HTTP_1_1,
			Headers: []Header{
				{"Host", "0.0.0.0=5000"},
				{"User-Agent", "Mozilla/5.0 (X11; U; Linux i686; en-US; rv:1.9) Gecko/2008061015 Firefox/3.0"},
				{"Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
				{"Accept-Language", "en-us,en;q=0.5"},
				{"Accept-Encoding", "gzip,deflate"},
				{"Accept-Charset", "ISO-8859-1,utf-8;q=0.7,*;q=0.7"},
				{"Keep-Alive", "300"},
				{"Connection", "keep-alive"},
			},
		},
		expected: "GET /favicon.ico HTTP/1.1\r\n" +
			"Host: 0.0.0.0=5000\r\n" +
			"User-Agent: Mozilla/5.0 (X11; U; Linux i686; en-US; rv:1.9) Gecko/2008061015 Firefox/3.0\r\n" +
			"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\n" +
			"Accept-Language: en-us,en;q=0.5\r\n" +
			"Accept-Encoding: gzip,deflate\r\n" +
			"Accept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.7\r\n" +
			"Keep-Alive: 300\r\n" +
			"Connection: keep-alive\r\n" +
			"\r\n",
	},
	{
		name: "dumbfuck",
		Request: Request{
			Method:  "GET",
			Path:    "/dumbfuck",
			Version: HTTP_1_1,
			Headers: []Header{
				{"aaaaaaaaaaaaa", "++++++++++"},
			},
		},
		expected: "GET /dumbfuck HTTP/1.1\r\n" +
			// modified as client.writer always formats headers canonically.
			"aaaaaaaaaaaaa: ++++++++++\r\n" +
			"\r\n",
	},
}

func TestRequest(t *testing.T) {
	for _, tt := range requestTests {
		var b bytes.Buffer
		c := NewClient(&b)
		err := c.WriteRequest(&tt.Request)
		if actual := b.String(); actual != tt.expected || err != tt.err {
			t.Errorf("%s: expected %q %v, got %q, %v", tt.name, tt.expected, tt.err, actual, err)
		}
	}
}
