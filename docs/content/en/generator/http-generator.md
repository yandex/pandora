---
title: HTTP generator
description: Configure HTTP generator
categories: [Generator]
tags: [generator, http]
weight: 9
---

Full http (http2) generator config

```yaml
gun:
  type: http
  target: '[hostname]:443'
  ssl: true                     # With type: http2, false means h2c (cleartext HTTP/2)
  response-code: http           # HTTP status by default; grpc is for type: http2
  connect-ssl: false            # If true, Pandora accepts any certificate presented by the server and any host name in that certificate. Default: false
  tls-handshake-timeout: 1s     # Maximum waiting time for a TLS handshake. Default: 1s
  disable-keep-alives: false    # If true, disables HTTP keep-alives. Default: false
  disable-compression: true     # If true, prevents the Transport from requesting compression with an "Accept-Encoding: gzip" request header. Default: true
  max-idle-conns: 0             # Maximum number of idle (keep-alive) connections across all hosts. Zero means no limit. Default: 0
  max-idle-conns-per-host: 2    # Controls the maximum idle (keep-alive) connections to keep per-host. Default: 2
  idle-conn-timeout: 90s        # Maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit. Default: 90s
  response-header-timeout: 0    # Amount of time to wait for a server's response headers after fully writing the request (including its body, if any). Zero means no timeout. Default: 0
  expect-continue-timeout: 1s   # Amount of time to wait for a server's first response headers after fully writing the request headers if the request has an "Expect: 100-continue" header. Zero means no timeout. Default: 1s
  shared-client:
    enabled: false              # If TRUE, the generator will use a common transport client for all instances
    client-number: 1            # The number of shared clients can be increased. The default is 1
  dial:
    timeout: 1s                 # TCP connect timeout. Default: 3s
    dns-cache: true             # Enable DNS cache, remember remote address on first try, and use it in the future. Default: true
    dual-stack: true            # IPv4 is tried soon if IPv6 appears to be misconfigured and hanging. Default: true
    fallback-delay: 300ms       # The amount of time to wait for IPv6 to succeed before falling back to IPv4. Default 300ms
    keep-alive: 120s            # Interval between keep-alive probes for an active network connection Default: 120s
  answlog:
    enabled: true
    path: ./answ.log
    filter: all             # all - all http codes, warning - log 4xx and 5xx, error - log only 5xx. Default: error
  auto-tag:
    enabled: true
    uri-elements: 2         # URI elements used to autotagging. Default: 2
    no-tag-only: true       # When true, autotagged only ammo that has no tag before. Default: true
  httptrace:
    dump: true              # calculate response bytes
    trace: true             # calculate different request stages: connect time, send time, latency, request bytes
```

## gRPC responses over HTTP/2 and h2c

When the ammo already contains a complete gRPC request, the HTTP/2 gun can interpret the response as gRPC without reflection or request encoding:

```yaml
pools:
  - id: grpc-over-h2c
    gun:
      type: http2
      target: localhost:8095
      ssl: false                 # h2c; true selects HTTP/2 over TLS
      response-code: grpc
      answlog:
        enabled: true
        path: ./answ.log
        filter: warning
        sampling:
          enabled: false         # Log every matching response
    ammo:
      type: raw
      file: ./grpc.raw
    result:
      type: phout
      destination: ./phout.log
    rps: {type: const, ops: 10, duration: 30s}
    startup: {type: once, times: 10}
```

`response-code` accepts `http` and `grpc`. It is optional: omitting it selects `http` and reports the HTTP status. The `grpc` value is supported only for `type: http2`. The ammo must contain a complete gRPC POST request, including headers and body; this option does not encode protobuf messages or enable retries.

In gRPC mode the gun reads `grpc-status` from final trailers, or from the terminal headers block for a trailers-only response. When it is missing, or the response `Content-Type` is not gRPC, the [standard HTTP-to-gRPC fallback](https://github.com/grpc/grpc/blob/master/doc/http-grpc-status-mapping.md) applies. The result's `proto_code` uses the [same gRPC-to-HTTP mapping as the built-in gRPC gun](./grpc-generator.md#mapping-response-codes): `grpc-status: 0` becomes `200`, and `14` becomes `503`. The original HTTP status and gRPC code origin remain visible in answlog. A non-`200` HTTP status takes precedence over `grpc-status`: HTTP `503` with `grpc-status: 0` cannot become a successful shot.

RTT ends at the final HTTP response headers, before reading the body. The result is reported after the response is fully read and trailers are available. Answlog records the original gRPC code, decoded `grpc-message`, `grpc-status-details-bin` (raw base64 and decoded `google.rpc.Status` when valid), HTTP status, request, and response headers/trailers. Answlog retains `grpc-message` and details even without `grpc-status`; with HTTP `200` and a gRPC `Content-Type`, invalid base64 details count as gRPC `INTERNAL` (`500`), even if `grpc-status` is missing. A body read failure logs the available HTTP metadata and the read error. The binary response body is not buffered for logging. The `warning` and `error` filters use original gRPC codes, like the built-in gRPC gun. An endless stream has no final result until it ends.

# References

- Best practices
  - [RPS per instance](best_practices/rps-per-instance.md)
  - [Shared client](best_practices/shared-client.md)
