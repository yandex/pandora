# Pandora

[![Release](https://github.com/yandex/pandora/actions/workflows/release.yml/badge.svg)](https://github.com/yandex/pandora/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/yandex/pandora.svg?style=flat-square)](https://github.com/yandex/pandora/releases)
[![Test](https://github.com/yandex/pandora/actions/workflows/test.yml/badge.svg)](https://github.com/yandex/pandora/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/yandex/pandora/badge.svg?precision=2)](https://app.codecov.io/gh/yandex/pandora)
![Code lines](https://sloc.xyz/github/yandex/pandora/?category=code)

[![PkgGoDev](https://pkg.go.dev/badge/github.com/yandex/pandora)](https://pkg.go.dev/github.com/yandex/pandora)
[![Go Report Card](https://goreportcard.com/badge/github.com/yandex/pandora)](https://goreportcard.com/report/github.com/yandex/pandora)
[![Join the chat at https://gitter.im/yandex/pandora](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/yandex/pandora?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

Pandora is a high-performance load generator in Go language. It has built-in HTTP(S) and HTTP/2 support and you can write your own load scenarios in Go, compiling them just before your test.

## Documentation
[Documentation](https://yandex.github.io/pandora/)

## How to start

### Binary releases
[Download](https://github.com/yandex/pandora/releases) available.

### Building from sources
We use go 1.11 modules.
If you build pandora inside $GOPATH, please make sure you have env variable `GO111MODULE` set to `on`.
```bash
git clone https://github.com/yandex/pandora.git
cd pandora
make deps
go install
```

Or let [Yandex.Tank](https://yandextank.readthedocs.io/en/latest/core_and_modules.html#pandora) make it easy for you.


## Extension points

You can write plugins with the next [extension points](https://github.com/progrium/go-extpoints):

You can also cross-compile for other arch/os:
```
GOOS=linux GOARCH=amd64 go build
```

### Running your tests
Run the binary with your config (see config examples at [examples](https://github.com/yandex/pandora/tree/develop/examples)):

```bash
# $GOBIN should be added to $PATH
pandora myconfig.yaml
```

Or use Pandora with [Yandex.Tank](https://yandextank.readthedocs.io/en/latest/core_and_modules.html#pandora) and
[Overload](https://overload.yandex.net).

## Changelog

Install https://github.com/miniscruff/changie

You can add changie completion to you favorite shell https://changie.dev/cli/changie_completion/

### Using

See https://changie.dev/guide/quick-start/

Show current version `changie latest`

Show next minor version `changie next minor`

Add new comments - `changie new` - and follow interface

Create changelog release file - `changie batch v0.5.21`

Same for next version - `changie batch $(changie next patch)`

Merge to main CHANGELOG.md file - `changie merge`
