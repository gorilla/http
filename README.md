# gorilla/http

A simple, safe and powerful HTTP client for the Go language.

Build Status: [![Build Status](https://drone.io/github.com/gorilla/http/status.png)](https://drone.io/github.com/gorilla/http/latest)

# Introduction

## Why do we need a new HTTP client, the Go standard library already has one ?

The Go net/http package is excellent. It is fast, efficient, gets the job done, and comes batteries 
included with every Go installation. But, at the same time the net/http package is a victim of its
own success. The Go 1 contract is defines many fields in the net/http types which are redundant, surplus
or wrong. 

Similarly the success of the net/http package has enshrined bugs which cannot be changed due to the growing
amount of software written to expect that behavior.

# # Client only

One acknowledged mistake of the net/http package is its reuse of core types between server and client implementations. 

At one level this is admirable, HTTP messages, requests and responses are more alike than they are different so it makes
good engineering sense to reuse their logic where possible. 

gorilla/http is a Client implementation only, which allows us to focus on a set of layered types which encapsulate the
request flow from the client point of view without compromise.

# Specific featues

This section addresses specific limitations of the net/http package, and discusses the gorilla/http alternatives.

## Timeouts

Timeouts are critically important. By dint of the Go 1 contract timeouts have been bolted on to the net/http implementation where possible. 
gorilla/http will go further and implement timeouts for as many operations as possible, connection, request send, reponse headers, response body, total request/response time, keepalive, etc.

## Closing Response Bodies

Forgetting to close a Response.Body is a continual problem for Gophers. It would be wonderful to create a client which does not 
require the respones body to be closed, however this appears impossible to marry with the idea of connection reuse and pooling.
Instead gorilla/http will address this in two ways
1. The high level functions in the http package do not return types that require closing, for example, http.Get(url string) returns a []byte and an error. The []byte 
contains the entire response body. This should be sufficient for many REST style http calls which exchange small messages.
2. A the http.Client layer, methods will return an io.ReadCloser, not a complicated Response type. This io.ReadCloser must be closed before falling out of scope otherwise the client will panic crashing the application.

## Connection rate limiting

Rate limiting in terms of number of total connections in use, number of connections to a particular site will be controllable. By default gorilla/http will only use a reasonable number of concurent connections. 

Gorilla/http has a strictly layered design where the high level gorilla/http pacakge is responsible for request composition and connection management and the lower level http/client package is strictly responsible for the http transaction and the lowest level wire format.

## Reliable DNS lookups

gorilla/http will use an alternative DNS resolver library to avoid the limitations of the system libc resolver library.

## Robustness and correctness

As a client only package gorilla/http has more flexibility to bias correctness over performance. Gorilla will always favor correctness of impemetantion over performance, and we believe this is the correct trade off. Having said that performance is a feature and gorilla/http will strive to keep its overheads compared to the underlying network transit cost as low as possible.

# Technical information

gorilla/http is divided into 4 layers. The top most layer is a set of convenience functions layerd on top of a
default client instance. These package level functions are intended to satisfy simple HTTP requests and only cover the most common verbs and use cases.

The next layer is http.Client which is a high level reusable http client, it transparently manages connection pooling and reuse and provides both common verbs and a 
general purpose Client.Do() interface for uncommon http verbs.

The lower layers are inside the http/client package and consist of types that deal with the abstract RFC2616 message form and marshal it on and off the wire. 

Interestingly, although these are the lowest level types, they do not deal with net.Conn implementations, but io.ReadWrite, connection setup, management and timeout control is handled by the owner of io.ReadWriter implementation passed to client.Client.
