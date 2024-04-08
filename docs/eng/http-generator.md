[Home](../index.md)

---

# HTTP generator

Full http (http2) generator config

```yaml
gun:
  type: http
  target: '[hostname]:443'
  ssl: true
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

# References

- Best practices
  - [RPS per instance](best_practices/rps-per-instance.md)
  - [Shared client](best_practices/shared-client.md)

---

[Home](../index.md)
