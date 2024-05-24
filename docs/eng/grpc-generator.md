[Home](index.md)

---

# gRPC generator

Full gRPC generator config

```yaml
gun:
  type: grpc
  target: '[hostname]:443'
  timeout: 15s              # Grpc request timeout. Default: 15s
  tls: false                # If true, Pandora accepts any certificate presented by the server and any host name in that certificate. Default: false
  reflect_port: 8000        # If your reflection service is located on a port other than the main server
  reflect_metadata:         # Separate metadata data for reflection service
    auth: Token
  shared-client:
    enabled: false          # If TRUE, the generator will use a common transport client for all instances
    client-number: 1        # The number of shared clients can be increased. The default is 1
  dial_options:
    authority: some.host    # Specifies the value to be used as the :authority pseudo-header and as the server name in authentication handshake
    timeout: 1s             # Timeout for dialing GRPC connect. Default: 1s
  answlog:
    enabled: true
    path: ./answ.log
    filter: all            # all - all http codes, warning - log 4xx and 5xx, error - log only 5xx. Default: error
```

## Mapping Response Codes

Pandora uses the gRPC client from google.golang.org/grpc as a client (https://github.com/grpc/grpc-go)

But to unify reports it converts them into HTTP codes.

### Mapping table gPRC StatusCode -> HTTP StatusCode

| gRPC Status Name   | gRPC Status Code | HTTP Status Code |
|--------------------|------------------|------------------|
| OK                 | 0                | 200              |
| Canceled           | 1                | 499              |
| InvalidArgument    | 3                | 400              |
| DeadlineExceeded   | 4                | 504              |
| NotFound           | 5                | 404              |
| AlreadyExists      | 6                | 409              |
| PermissionDenied   | 7                | 403              |
| ResourceExhausted  | 8                | 429              |
| FailedPrecondition | 9                | 400              |
| Aborted            | 10               | 409              |
| OutOfRange         | 11               | 400              |
| Unimplemented      | 12               | 501              |
| Unavailable 14     | 14               | 503              |
| Unauthenticated 16 | 16               | 401              |
| unknown            | -                | 500              |


# References

- [Scenario generator / gRPC](scenario-grpc-generator.md)
- Best practices
  - [RPS per instance](best_practices/rps-per-instance.md)
  - [Shared client](best_practices/shared-client.md)

---

[Home](index.md)
