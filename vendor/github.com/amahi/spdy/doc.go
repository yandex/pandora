// Amahi SPDY is a library built from scratch in the "Go way" for building SPDY clients and servers in the Go programming language.
//
// It supports a subset of SPDY 3.1 http://www.chromium.org/spdy/spdy-protocol/spdy-protocol-draft3-1
//
// Check the source code, examples and overview at https://github.com/amahi/spdy
//
// This library is used in a streaming server/proxy implementation for Amahi, the [home and media server](https://www.amahi.org).
//
// Goals
//
// The goals are reliability, streaming and performance/scalability.
//
// 1) Design for reliability means that network connections are assumed to disconnect at any time, especially when it's most inapropriate for the library to handle. This also includes potential issues with bugs in within the library, so the library tries to handle all crazy errors in the most reasonable way. A client or a server built with this library should be able to run for months and months of reliable operation. It's not there yet, but it will be.
//
// 2) Streaming requests, unlike typical HTTP requests (which are short), require working with an arbitrary large number of open requests (streams) simultaneously, and most of them are flow-constrained at the client endpoint. Streaming clients kind of misbehave too, for example, they open and close many streams rapidly with Range request to check certain parts of the file. This is common with endpoint clients like VLC or Quicktime (Safari on iOS or Mac OS X). We wrote this library with the goal of making it not just suitable for HTTP serving, but also for streaming.
//
// 3) The library was built with performance and scalability in mind, so things have been done using as little blocking and copying of data as possible. It was meant to be implemented in the "go way", using concurrency extensively and channel communication. The library uses mutexes very sparingly so that handling of errors at all manner of inapropriate times becomes easier. It goes to great lengths to not block, establishing timeouts when network and even channel communication may fail. The library should use very very little CPU, even in the presence of many streams and sessions running simultaneously.
package spdy
