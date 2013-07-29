package client

// These tests are adapted from the node.js/http_parser tests suite
// https://github.com/joyent/node/blob/master/deps/http_parser/test.c
// http_parser is listed as a MIT compatible licence.

import (
	"bufio"
	"bytes"
	"io"
	"reflect"
	"strings"
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
	{
		name: "fragment in uri",
		Request: Request{
			Method:  "GET",
			Path:    "/forums/1/topics/2375",
			Query:   []string{"page=1"},
			Version: HTTP_1_1,
		},
		// modified, sending a fragment sounds wrong.
		expected: "GET /forums/1/topics/2375?page=1 HTTP/1.1\r\n" +
			"\r\n",
	},
	{
		name: "get no headers no body",
		Request: Request{
			Method:  "GET",
			Path:    "/get_no_headers_no_body/world",
			Version: HTTP_1_1,
		},
		expected: "GET /get_no_headers_no_body/world HTTP/1.1\r\n" +
			"\r\n",
	},
	{
		name: "get one header no body",
		Request: Request{
			Method:  "GET",
			Path:    "/get_one_header_no_body",
			Version: HTTP_1_1,
			Headers: []Header{{"Accept", "*/*"}},
		},
		expected: "GET /get_one_header_no_body HTTP/1.1\r\n" +
			"Accept: */*\r\n" +
			"\r\n",
	},
	/**
		// SendRequest supplies a content length
		{
			name: "get funky content length body hello",
			Request: Request{
				Method:  "GET",
				Path:    "/get_funky_content_length_body_hello",
				Version: HTTP_1_0,
				Headers: []Header{{"conTENT-Length", "5"}},
				Body:    strings.NewReader("HELLO"),
			},
			expected: "GET /get_funky_content_length_body_hello HTTP/1.0\r\n" +
				"conTENT-Length: 5\r\n" +
				"\r\n" +
				"HELLO",
		},
	**/
	{
		name: "post identity body world",
		Request: Request{
			Method:  "POST",
			Path:    "/post_identity_body_world",
			Query:   []string{"q=search"},
			Version: HTTP_1_1,
			Headers: []Header{
				{"Accept", "*/*"},
				{"Transfer-Encoding", "identity"},
				// SendRequest supplies a content length by default
				//	{"Content-Length", "5"},
			},
			Body: strings.NewReader("World"),
		},
		// modified to remove the fragment
		expected: "POST /post_identity_body_world?q=search HTTP/1.1\r\n" +
			"Accept: */*\r\n" +
			"Transfer-Encoding: identity\r\n" +
			"Content-Length: 5\r\n" +
			"\r\n" +
			"World",
	},
	{
		name: "post - chunked body: all your base are belong to us",
		Request: Request{
			Method:  "POST",
			Path:    "/post_chunked_all_your_base",
			Version: HTTP_1_1,
			Headers: []Header{
			// SendRequest handles this for us
			// {"Transfer-Encoding", "chunked"},
			},
			Body: b("all your base are belong to us"),
		},
		expected: "POST /post_chunked_all_your_base HTTP/1.1\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"\r\n" +
			"1e\r\nall your base are belong to us\r\n" +
			"0\r\n",
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

var responseTests = []struct {
	name string
	data string
	Response
	err error
}{
	{
		name: "google 301",
		data: "HTTP/1.1 301 Moved Permanently\r\n" +
			"Location: http://www.google.com/\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Date: Sun, 26 Apr 2009 11:11:49 GMT\r\n" +
			"Expires: Tue, 26 May 2009 11:11:49 GMT\r\n" +
			"X-$PrototypeBI-Version: 1.6.0.3\r\n" +
			"Cache-Control: public, max-age=2592000\r\n" +
			"Server: gws\r\n" +
			"Content-Length:  219  \r\n" +
			"\r\n" +
			"<HTML><HEAD><meta http-equiv=\"content-type\" content=\"text/html;charset=utf-8\">\n" +
			"<TITLE>301 Moved</TITLE></HEAD><BODY>\n" +
			"<H1>301 Moved</H1>\n" +
			"The document has moved\n" +
			"<A HREF=\"http://www.google.com/\">here</A>.\r\n" +
			"</BODY></HTML>\r\n",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{301, "Moved Permanently"},
			Headers: []Header{
				{"Location", "http://www.google.com/"},
				{"Content-Type", "text/html; charset=UTF-8"},
				{"Date", "Sun, 26 Apr 2009 11:11:49 GMT"},
				{"Expires", "Tue, 26 May 2009 11:11:49 GMT"},
				{"X-$PrototypeBI-Version", "1.6.0.3"},
				{"Cache-Control", "public, max-age=2592000"},
				{"Server", "gws"},
				// {"Content-Length", "219  "},
				// TODO(dfc) should trailing whitespace be preserved?
				{"Content-Length", "219"},
			},
			Body: strings.NewReader("<HTML><HEAD><meta http-equiv=\"content-type\" content=\"text/html;charset=utf-8\">\n" +
				"<TITLE>301 Moved</TITLE></HEAD><BODY>\n" +
				"<H1>301 Moved</H1>\n" +
				"The document has moved\n" +
				"<A HREF=\"http://www.google.com/\">here</A>.\r\n" +
				"</BODY></HTML>\r\n"),
		},
	},
	{
		name: "no content-length response",
		data: "HTTP/1.1 200 OK\r\n" +
			"Date: Tue, 04 Aug 2009 07:59:32 GMT\r\n" +
			"Server: Apache\r\n" +
			"X-Powered-By: Servlet/2.5 JSP/2.1\r\n" +
			"Content-Type: text/xml; charset=utf-8\r\n" +
			"Connection: close\r\n" +
			"\r\n" +
			"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
			"<SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\">\n" +
			"  <SOAP-ENV:Body>\n" +
			"    <SOAP-ENV:Fault>\n" +
			"       <faultcode>SOAP-ENV:Client</faultcode>\n" +
			"       <faultstring>Client Error</faultstring>\n" +
			"    </SOAP-ENV:Fault>\n" +
			"  </SOAP-ENV:Body>\n" +
			"</SOAP-ENV:Envelope>",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{200, "OK"},
			Headers: []Header{
				{"Date", "Tue, 04 Aug 2009 07:59:32 GMT"},
				{"Server", "Apache"},
				{"X-Powered-By", "Servlet/2.5 JSP/2.1"},
				{"Content-Type", "text/xml; charset=utf-8"},
				{"Connection", "close"},
			},
			Body: strings.NewReader("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
				"<SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\">\n" +
				"  <SOAP-ENV:Body>\n" +
				"    <SOAP-ENV:Fault>\n" +
				"       <faultcode>SOAP-ENV:Client</faultcode>\n" +
				"       <faultstring>Client Error</faultstring>\n" +
				"    </SOAP-ENV:Fault>\n" +
				"  </SOAP-ENV:Body>\n" +
				"</SOAP-ENV:Envelope>"),
		},
	},
	{
		name: "404 no headers no body",
		data: "HTTP/1.1 404 Not Found\r\n\r\n",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{404, "Not Found"},
		},
	},
	{
		name: "301 no response phrase",
		data: "HTTP/1.1 301\r\n\r\n",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{Code: 301},
		},
	},
	{
		name: "200 trailing space on chunked body",
		data: "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"\r\n" +
			"25  \r\n" +
			"This is the data in the first chunk\r\n" +
			"\r\n" +
			"1C\r\n" +
			"and this is the second one\r\n" +
			"\r\n" +
			"0  \r\n" +
			"\r\n",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{200, "OK"},
			Headers: []Header{
				{"Content-Type", "text/plain"},
				{"Transfer-Encoding", "chunked"},
			},
			Body: strings.NewReader("This is the data in the first chunk\r\nand this is the second one\r\n"),
		},
	},
	{
		name: "no carriage ret",
		data: "HTTP/1.1 200 OK\n" +
			"Content-Type: text/html; charset=utf-8\n" +
			"Connection: close\n" +
			"\n" +
			"these headers are from http://news.ycombinator.com/",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{200, "OK"},
			Headers: []Header{
				{"Content-Type", "text/html; charset=utf-8"},
				{"Connection", "close"},
			},
			Body: strings.NewReader("these headers are from http://news.ycombinator.com/"),
		},
	},
	{
		name: "proxy connection",
		data: "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Content-Length: 11\r\n" +
			"Proxy-Connection: close\r\n" +
			"Date: Thu, 31 Dec 2009 20:55:48 +0000\r\n" +
			"\r\n" +
			"hello world",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{200, "OK"},
			Headers: []Header{
				{"Content-Type", "text/html; charset=UTF-8"},
				{"Content-Length", "11"},
				{"Proxy-Connection", "close"},
				{"Date", "Thu, 31 Dec 2009 20:55:48 +0000"},
			},
			Body: b("hello world"),
		},
	},
	{
		name: "underscore header key",
		data: "HTTP/1.1 200 OK\r\n" +
			"Server: DCLK-AdSvr\r\n" +
			"Content-Type: text/xml\r\n" +
			"Content-Length: 0\r\n" +
			"DCLK_imp: v7;x;114750856;0-0;0;17820020;0/0;21603567/21621457/1;;~okv=;dcmt=text/xml;;~cs=o\r\n\r\n",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{200, "OK"},
			Headers: []Header{
				{"Server", "DCLK-AdSvr"},
				{"Content-Type", "text/xml"},
				{"Content-Length", "0"},
				{"DCLK_imp", "v7;x;114750856;0-0;0;17820020;0/0;21603567/21621457/1;;~okv=;dcmt=text/xml;;~cs=o"},
			},
		},
	},
	{
		name: "bonjourmadame.fr",
		data: "HTTP/1.0 301 Moved Permanently\r\n" +
			"Date: Thu, 03 Jun 2010 09:56:32 GMT\r\n" +
			"Server: Apache/2.2.3 (Red Hat)\r\n" +
			"Cache-Control: public\r\n" +
			"Pragma: \r\n" +
			"Location: http://www.bonjourmadame.fr/\r\n" +
			"Vary: Accept-Encoding\r\n" +
			"Content-Length: 0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Connection: keep-alive\r\n" +
			"\r\n",
		Response: Response{
			Version: HTTP_1_0,
			Status:  Status{301, "Moved Permanently"},
			Headers: []Header{
				{"Date", "Thu, 03 Jun 2010 09:56:32 GMT"},
				{"Server", "Apache/2.2.3 (Red Hat)"},
				{"Cache-Control", "public"},
				{"Pragma", ""},
				{"Location", "http://www.bonjourmadame.fr/"},
				{"Vary", "Accept-Encoding"},
				{"Content-Length", "0"},
				{"Content-Type", "text/html; charset=UTF-8"},
				{"Connection", "keep-alive"},
			},
		},
	},
	{
		name: "field underscore",
		data: "HTTP/1.1 200 OK\r\n" +
			"Date: Tue, 28 Sep 2010 01:14:13 GMT\r\n" +
			"Server: Apache\r\n" +
			"Cache-Control: no-cache, must-revalidate\r\n" +
			"Expires: Mon, 26 Jul 1997 05:00:00 GMT\r\n" +
			".et-Cookie: PlaxoCS=1274804622353690521; path=/; domain=.plaxo.com\r\n" +
			"Vary: Accept-Encoding\r\n" +
			"_eep-Alive: timeout=45\r\n" + /* semantic value ignored */
			"_onnection: Keep-Alive\r\n" + /* semantic value ignored */
			"Transfer-Encoding: chunked\r\n" +
			"Content-Type: text/html\r\n" +
			"Connection: close\r\n" +
			"\r\n" +
			"0\r\n\r\n",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{200, "OK"},
			Headers: []Header{
				{"Date", "Tue, 28 Sep 2010 01:14:13 GMT"},
				{"Server", "Apache"},
				{"Cache-Control", "no-cache, must-revalidate"},
				{"Expires", "Mon, 26 Jul 1997 05:00:00 GMT"},
				{".et-Cookie", "PlaxoCS=1274804622353690521; path=/; domain=.plaxo.com"},
				{"Vary", "Accept-Encoding"},
				{"_eep-Alive", "timeout=45"},
				{"_onnection", "Keep-Alive"},
				{"Transfer-Encoding", "chunked"},
				{"Content-Type", "text/html"},
				{"Connection", "close"},
			},
		},
	},
	{
		name: "non-ASCII in status line",
		data: "HTTP/1.1 500 Oriëntatieprobleem\r\n" +
			"Date: Fri, 5 Nov 2010 23:07:12 GMT+2\r\n" +
			"Content-Length: 0\r\n" +
			"Connection: close\r\n" +
			"\r\n",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{500, "Oriëntatieprobleem"},
			Headers: []Header{
				{"Date", "Fri, 5 Nov 2010 23:07:12 GMT+2"},
				{"Content-Length", "0"},
				{"Connection", "close"},
			},
		},
	},
	{
		name: "http version 0.9",
		data: "HTTP/0.9 200 OK\r\n" +
			"\r\n",
		Response: Response{
			Version: Version{0, 9},
			Status:  Status{200, "OK"},
		},
	},
	{
		name: "neither content-length nor transfer-encoding response",
		data: "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain\r\n" +
			"\r\n" +
			"hello world",
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{200, "OK"},
			Headers: []Header{{"Content-Type", "text/plain"}},
			Body:    b("hello world"),
		},
	},
	{
		name: "field space",
		data: "HTTP/1.1 200 OK\r\n" +
			"Server: Microsoft-IIS/6.0\r\n" +
			"X-Powered-By: ASP.NET\r\n" +
			"en-US Content-Type: text/xml\r\n" + /* this is the problem */
			"Content-Type: text/xml\r\n" +
			"Content-Length: 16\r\n" +
			"Date: Fri, 23 Jul 2010 18:45:38 GMT\r\n" +
			"Connection: keep-alive\r\n" +
			"\r\n" +
			"<xml>hello</xml>", /* fake body */
		Response: Response{
			Version: HTTP_1_1,
			Status:  Status{200, "OK"},
			Headers: []Header{
				{"Server", "Microsoft-IIS/6.0"},
				{"X-Powered-By", "ASP.NET"},
				{"en-US Content-Type", "text/xml"},
				{"Content-Type", "text/xml"},
				{"Content-Length", "16"},
				{"Date", "Fri, 23 Jul 2010 18:45:38 GMT"},
				{"Connection", "keep-alive"},
			},
			Body: b("<xml>hello</xml>"),
		},
	},
}

func TestResponse(t *testing.T) {
	for _, tt := range responseTests {
		client := &client{reader: reader{Reader: bufio.NewReader(strings.NewReader(tt.data))}}
		resp, err := client.ReadResponse()
		t.Log(tt.name)
		if !sameErr(err, tt.err) {
			t.Errorf("client.ReadResponse(%q): err expected %v, got %v", tt.data, tt.err, err)
			continue
		}
		if resp.Version != tt.Response.Version || resp.Status != tt.Response.Status {
			t.Errorf("client.ReadResponse(%q): Version/Status expected %q %q, got %q %q", tt.data, tt.Response.Version, tt.Response.Status, resp.Version, resp.Status)
			continue
		}
		if !reflect.DeepEqual(tt.Response.Headers, resp.Headers) || err != tt.err {
			t.Errorf("client.ReadResponse(%q): Headers expected %v %v, got %v %v", tt.data, tt.Response.Headers, tt.err, resp.Headers, err)
			continue
		}
		if actual, expected := resp.CloseRequested(), tt.Response.CloseRequested(); actual != expected {
			t.Errorf("client.ReadResponse(%q): CloseRequested expected %v, got %v", tt.data, expected, actual)
			continue
		}
		var buf bytes.Buffer
		var expected, actual string
		if tt.Response.Body != nil {
			_, err = io.Copy(&buf, tt.Response.Body)
			expected = buf.String()
		}
		buf.Reset()
		if resp.Body != nil {
			_, err = io.Copy(&buf, resp.Body)
			actual = buf.String()
		}
		if actual != expected || err != tt.err {
			t.Errorf("client.ReadResponse(%q): expected %q %v, got %q %v", tt.data, expected, tt.err, actual, err)
		}
	}
}
