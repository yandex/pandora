---
title: HTTP Ammo providers
description: HTTP Ammo provider is a source of test data - it makes ammo object
categories: [Provider]
tags: [provider, http]
weight: 10
---

HTTP Ammo provider is a source of test data: it makes ammo object.

There is a common rule for any (built-in) provider: data supplied by ammo provider are records that will be pushed via
established connection to external host (defined in pandora config via _pool.gun.target_ option). Thus, you cannot
define in the ammofile to which _physical_ host your ammo will be sent.

## Test data

### http/json

jsonline format, 1 row — 1 json-encoded ammo.

Pay attention to special header _Host_ defined `outside` of Headers dictionary.

_Host_ inside Headers section will be silently ignored.

Ammofile sample:

```
{"tag": "tag1", "uri": "/", "method": "GET", "headers": {"Accept": "*/*", "Accept-Encoding": "gzip, deflate", "User-Agent": "Pandora"}, "host": "example.com"}
```

Config sample:

```yaml
pools:
  - ammo:
      type: http/json                # ammo format
      file: ./ammofile               # ammo file path
```

If the lines in a ammo file are more than 64 KB in length, specify the buffer size for reading lines using the `maxammosize` parameter. The value is specified in bytes and must be greater than the longest string. Configuration example:

```yaml
pools:
  - ammo:
      type: uri             # ammo format
      file: ./ammofile      # ammo file path
      maxammosize: 1000000  # maximum ammo size
```

### raw (request-style)

Raw HTTP request format. If you like to use _telnet_ firing HTTP requests, you'll love this.
Also known as phantom's _request-style_.

File contains size-prefixed HTTP requests. Each ammo is prefixed with a header line (delimited with \n), which consists
of two fields delimited by a space: ammo size and tag. Ammo size is in bytes (integer, including special characters like
CR, LF). Tag is a string. You can read about this format (with detailed instructions) at
[Yandex.Tank documentation](https://yandextank.readthedocs.io/en/latest/tutorial.html#request-style)

Ammofile sample:

```
73 good
GET / HTTP/1.0
Host: xxx.tanks.example.com
User-Agent: xxx (shell 1)

77 bad
GET /abra HTTP/1.0
Host: xxx.tanks.example.com
User-Agent: xxx (shell 1)

78 unknown
GET /ab ra HTTP/1.0
Host: xxx.tanks.example.com
User-Agent: xxx (shell 1)
```

Config sample:

```yaml
pools:
  - ammo:
      type: raw                      # ammo format
      file: ./ammofile               # ammo file path
```

You can define common headers using special config option `headers`. Headers in ammo file have priority. Format: list of
strings.

Example:

```yaml
pools:
  - ammo:
      type: raw                      # ammo format
      file: ./ammofile               # ammo file path
      headers:
        - "[Host: yourhost.tld]"
        - "[User-Agent: some user agent]"
```

### uri-style

List of URIs and headers

Ammofile sample:

```
[Connection: close]
[Host: your.host.tld]
[Cookie: None]
/?drg tag1
/
/buy tag2
[Cookie: test]
/buy/?rt=0&station_to=7&station_from=9
```

Config sample:


```yaml
pools:
  - ammo:
      type: uri                      # ammo format
      file: ./ammofile               # ammo file path
```

You can define common headers using special config option `headers`. Headers in ammo file have priority. Format: list of
strings.

Example:

```yaml
pools:
  - ammo:
      type: uri                      # ammo format
      file: ./ammofile               # ammo file path
      headers:
        - "[Host: yourhost.tld]"
        - "[User-Agent: some user agent]"
```

If the lines in a ammo file are more than 64 KB in length, specify the buffer size for reading lines using the `maxammosize` parameter. The value is specified in bytes and must be greater than the longest string.  Configuration example:

```yaml
pools:
  - ammo:
      type: uri             # ammo format
      file: ./ammofile      # ammo file path
      maxammosize: 1000000  # maximum ammo size
```

## Features

### Ammo filters

Each http ammo provider lets you choose specific ammo for your test from ammo file with _chosencases_ setting:

```yaml
pools:
  - ammo:
      type: uri                        # ammo format
      chosencases: [ "tag1", "tag2" ]  # use only "tag1" and "tag2" ammo for this test
      file: ./ammofile                 # ammo file path
```

Tags are defined in ammo files as shown below:

#### http/json:

```
{"tag": "tag1", "uri": "/",
```

#### raw (request-style):

```
73 tag1
GET / HTTP/1.0
```

#### uri-style:

```
/?drg tag1
/
/buy tag2
```

### HTTP Ammo middlewares

HTTP Ammo providers have the ability to modify HTTP request just before execution.
Middlewares are used for this purpose. An example of Middleware that sets the Date header in a request.

```yaml
pools:
  - ammo:
      type: uri
      ...
      middlewares:
        - type: header/date
          location: EST
          headerName: Date
```

List of built-in HTTP Ammo middleware:

- header/date

You can create your own middleware. But in order to do that you need to register them in [custom pandora](generator/custom.md)

```go
import (
    "github.com/yandex/pandora/components/providers/http/middleware"
    "github.com/yandex/pandora/components/providers/http/middleware/headerdate"
    httpRegister "github.com/yandex/pandora/components/providers/http/register"
)

httpRegister.HTTPMW("header/date", func (cfg headerdate.Config) (middleware.Middleware, error) {
    return headerdate.NewMiddleware(cfg)
})
```

For more on how to write custom pandora, see [Custom](generator/custom.md).

### HTTP Ammo preloaded

Pandora's architecture is designed for high performance. To achieve high performance, Pandora prepares ammo for each
instance.

If you have **large requests** and **a large number of instances**, Pandora starts using a lot of memory.

For this case HTTP providers has a `preload` flag. If it's set to `true`, the provider will load the ammo file into
memory and use the body of the request from memory

Example:

```yaml
pools:
  - ammo:
      type: ...
      ...
      preload: true
```

### Limiting the number of ammos

By default, all providers start reading from the beginning when they reach the end of the ammos.
The `passes` parameter is used to limit the number of repetitions.

If you need to set a general limit on the number of ammos that a provider can issue, you should use the `limit` parameter.

Example config:

```yaml
pools:
  - ammo:
      type: http/json     # ammo format
      file: ./ammo.json   # ammo file path
      limit: 0            # Limit limits total num of ammo. Unlimited if zero. Default: 0
      passes: 0           # Passes limits ammo file passes. Unlimited if zero. Default: 0
```
