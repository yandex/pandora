[Home](../index.md)

---

# HTTP Ammo providers

- [Test data](#test-data)
  - [http/json](#httpjson)
  - [raw (request-style)](#raw-request-style)
  - [uri-style](#uri-style)
- [Features](#features)
  - [Ammo filters](#ammo-filters)
  - [HTTP Ammo middlewares](#http-ammo-middlewares)
  - [HTTP Ammo preloaded](#http-ammo-preloaded)

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

You can create your own middleware. But in order to do that you need to register them in custom pandora

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

For more on how to write custom pandora, see [Custom](custom.md).

### HTTP Ammo preloaded

Pandora's architecture is designed for high performance. To achieve high performance, Pandora prepares ammo for each
instance.

If you have **large requests** and **a large number of instances**, Pandora starts using a lot of memory.

For this case HTTP providers has a ``preload`` flag. If it's set to ``true``, the provider will load the ammo file into
memory and use the body of the request from memory

Example:

```yaml
pools:
  - ammo:
      type: ...
      ...
      preload: true
```

---

[Home](../index.md)
