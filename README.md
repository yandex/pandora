# Pandora

[![Join the chat at https://gitter.im/yandex/pandora](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/yandex/pandora?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Build Status](https://travis-ci.org/yandex/pandora.svg)](https://travis-ci.org/yandex/pandora)
[![Coverage Status](https://coveralls.io/repos/yandex/pandora/badge.svg?branch=master&service=github)](https://coveralls.io/github/yandex/pandora?branch=master)

A load generator in Go language.

## Install
Compile a binary with go tool (use go >= 1.5.2):
```
go get github.com/yandex/pandora
go build github.com/yandex/pandora
```

There are also [binary releases](https://github.com/yandex/pandora/releases) available.

Run this binary with your .json config (see [examples](https://github.com/yandex/pandora/tree/master/example/config)):
```
./pandora myconfig.json
```
Or let [Yandex.Tank](https://yandextank.readthedocs.io/en/latest/core_and_modules.html#pandora) make it easy for you.


## Extension points

You can write plugins with the next [extension points](https://github.com/progrium/go-extpoints):

```
ammo.Provider
aggregate.ResultListener
limiter.Limiter
gun.Gun
```

## Build tags

If you don want to build pandora without http gun:
```
go build -tags 'noHttpGun' github.com/yandex/pandora
```

If you don want to build pandora without spdy gun:
```
go build -tags 'noSpdyGun' github.com/yandex/pandora
```

## Basic concepts

### Architectural scheme

See architectural scheme source in ```docs/architecture.graphml```. It was created with
[YeD](https://www.yworks.com/en/products/yfiles/yed/) editor, so you’ll probably
need it to open the file.

![Architectural scheme](/docs/architecture.png)

Pandora is made of components. Components talk to each other over the channels. There are different types of components.

### Component types

#### Ammo provider and decoder

Ammo decoder knows how to make an ammo object from an ammo file or other external resource. Ammo provider uses a decoder
to decode ammo and passes ammo objects to the Users.

#### User pool

User pool controls the creation of Users. All users from one user pool will get ammo from one ammo provider. User creation
schedule is controlled with Startup Limiter. All users from one user pool will also have guns of the same type.

#### Limiters

Limiters are objects that will put messages to its underlying channel according to a schedule. User creation, shooting or
other processes thus can be controlled by a Limiter.

#### Users and Guns
User takes an ammo, waits for a limiter tick and then shoots with a Gun it has. Guns are the tools to send a request to your
service and measure the parameters of the response.

#### Result Listener
Result listener's task is to collect measured samples and save them somewhere.
