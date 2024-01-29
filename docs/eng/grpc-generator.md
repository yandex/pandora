[Home](index.md)

---

# gRPC generator

Full gRPC generator config

```yaml
gun:
  type: http
  target: '[hostname]:443'
  timeout: 15s
  tls: true
  dial_options:
    timeout: 1s
    authority: string
  answlog:
    enabled: true
    path: ./answ.log
    filter: all            # all - all http codes, warning - log 4xx and 5xx, error - log only 5xx. Default: error
```

---

[Home](index.md)
