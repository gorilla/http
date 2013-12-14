# gorilla/http

A simple, safe and powerful HTTP client for the Go language.

Build Status: [![Build Status](https://travis-ci.org/gorilla/http.png?branch=master)](https://travis-ci.org/gorilla/http)

# Image of an under construction GIF

This project is experimental. We welcome contributors and early adopters if you're feeling brave.

# Introduction

## Why does Go need a new HTTP client, the standard library already has one ?

The Go `net/http` package is excellent. It is fast, efficient, gets the job done, and comes batteries
included with every Go installation. At the same time the `net/http` package is a victim of its own
success. The Go 1 contract defines many fields in the `net/http` types which are redundant or surplus.

Similarly the success of the `net/http` package has enshrined bugs which cannot be changed due to the
growing amount of software written to expect that behaviour.

## Client only

One acknowledged shortcoming of the `net/http` package is its reuse of core types between server and client implementations.

At one level this is admirable, HTTP messages; requests and responses, are more alike than they are different so it
makes good engineering sense to reuse their logic where possible. However, combined with the Go 1 contract, this has
lead to compromises.

`gorilla/http` is a client implementation only. This allows us to focus on a set of layered types which encapsulate the
complete request flow from the client point of view without compromise.

# Specific features

This section addresses specific limitations of the `net/http` package and discusses the `gorilla/http` alternatives.

## Timeouts

Timeouts are critically important. By dint of the Go 1 contract, timeouts have been bolted on to the `net/http`
implementation where possible. `gorilla/http` will go further and implement timeouts for as many operations as
possible; connection, request send, response headers, response body, total request/response time, keepalive, etc.

## Closing Response Bodies

Forgetting to close a `Response.Body` is a continual problem for Gophers. It would be wonderful to create a
client which does not require the response body to be closed, however this appears impossible to marry with
the idea of connection reuse and pooling.

Instead `gorilla/http` will address this in two ways
 1. The high level functions in the `gorilla/http` package do not return types that require closing. For example,
`gorilla/http.Get(w io.Writer, url string)` mirrors the interface of `io.Copy` and should be sufficient for many
REST style http calls which exchange small messages.
 2. At the `http.Client` layer, methods will return an `io.ReadCloser`, not a complex `Response` type. This
`io.ReadCloser` *must* be closed before falling out of scope otherwise the client will panic the application.

## Connection rate limiting

Rate limiting in terms of number of total connections in use, number of connections to a particular site will
be controllable. By default `gorilla/http` will only use a reasonable number of concurrent connections.

`gorilla/http` has a strictly layered design where the high level `gorilla/http` package is responsible for
request composition and connection management and the lower level `gorilla/http/client` package is strictly
responsible for the http transaction and the lowest level wire format.

## Reliable DNS lookups

`gorilla/http` will use an alternative DNS resolver library to avoid the limitations of the system libc resolver library.

## Robustness and correctness

As a client only package, `gorilla/http` has flexibility to bias correctness over performance. Gorilla will always
favor correctness of implementation over performance, and we believe this is the correct trade off. Having said that
performance is a feature and `gorilla/http` strives to keep its overheads compared to the underlying network transit
cost as low as possible.

# Roadmap

The roadmap for the project is captured as open issues in GitHub.

# Contributions and feedback

Please raise issues and suggestions on the GitHub project page, <https://github.com/gorilla/http/issues>.

Questions and discussion can also be directed to the general Gorilla mailing list <https://groups.google.com/group/gorilla-web>.

# Technical information

`gorilla/http` is divided into 4 layers. The topmost layer is a set of convenience functions layered on top of a
default `gorilla/http.Client` instance. These package level functions are intended to satisfy simple HTTP requests
and only cover the most common verbs and use cases.

The next layer is `gorilla/http.Client` which is a high level reusable HTTP client. It transparently manages connection
pooling and reuse and provides both common verbs and a general purpose `Client.Do()` interface for uncommon http verbs.

The lower layers are inside the `gorilla/http/client` package and consist of types that deal with the abstract RFC2616
message form and marshal it on and off the wire.

Interestingly, although these are the lowest level types, they do not deal with `net.Conn` implementations, but
`io.ReadWriter`, connection setup, management and timeout control is handled by the owner of the `io.ReadWriter`
implementation passed to `client.Client`.
