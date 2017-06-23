# Pandora

[![Join the chat at https://gitter.im/yandex/pandora](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/yandex/pandora?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Build Status](https://travis-ci.org/yandex/pandora.svg)](https://travis-ci.org/yandex/pandora)
[![Coverage Status](https://coveralls.io/repos/yandex/pandora/badge.svg?branch=master&service=github)](https://coveralls.io/github/yandex/pandora?branch=master)

A load generator in Go language.

## How to start

### Binary releases
[Download](https://github.com/yandex/pandora/releases) available.

### Building from sources
We use [glide](https://glide.sh) for package management. Install it before compiling Pandora
Compile a binary with go tool (use go >= 1.8):
```bash
go get github.com/yandex/pandora
glide install github.com/yandex/pandora
go build github.com/yandex/pandora
```

### Running your tests
Run the binary with your config (see config examples at [examples](https://github.com/yandex/pandora/tree/master/cli/config)):

```bash
# $GOBIN should be added to $PATH
pandora myconfig.yaml
```

Or use Pandora with [Yandex.Tank](http://yandextank.readthedocs.org/en/latest/configuration.html#pandora) and
[Overload](https://overload.yandex.net).


## Example

`load.yaml`:

```yaml
pools:
  - id: HTTP pool                    # Pool name
    gun:
      type: http                     # Gun type
      target: example.com:80         # Gun target
    ammo:
      type: uri                      # Ammo format                        
      file: ./ammo.uri               # Ammo File
    result:
      type: phout                    # Report format (phout is for Yandex.Tank)
      destination: ./http_phout.log  # Report file name

    rps:                             # RPS Schedule
      type: periodic                 # shoot periodically
      period: 0.1s                   # ten batches each second
      max: 30                        # thirty batches total
      batch: 2                       # in batches of two shoots

    startup:                         # Startup Schedule
      type: periodic                 # start Instances periodically
      period: 0.5s                   # every 0.5 seconds
      batch: 1                       # one Instance at a time
      max: 5                         # five Instances total
```

`ammo.uri`:

```
/my/first/url
/my/second/url
```

Run your tests:

```
pandora load.yaml
```

The results are in `http_phout.log`. Use [Yandex.Tank](http://yandextank.readthedocs.org/en/latest/configuration.html#pandora)
and [Overload](https://overload.yandex.net) to plot them.

## Basic concepts

### Architectural scheme

See architectural scheme source in ```docs/architecture.graphml```. It was created with
[YeD](https://www.yworks.com/en/products/yfiles/yed/) editor.

![Architectural scheme](/docs/architecture.png)

Pandora is a set of components talking to each other through the channels. There are different types of components.

### Component types

#### Ammo Provider

Ammo Provider knows how to make an ammo object from an ammo file or other external resource. Instances get ammo objects
from Ammo Provider.

#### Instances Pool

Instances Pool manages the creation of Instances. You can think of one Instance as a single user that sends requests to
a server sequentially. All Instances from one Instances Pool get their ammo from one Ammo Provider. Instances creation
times are controlled by Startup Scheduler. All Instances from one Instances Pool also have Guns of the same type.

#### Scheduler

Scheduler controls other events' times by pushing messages to its underlying channel according to the Schedule.
It can control Instances startup times, RPS amount (requests per second) or other processes.

By combining two types of Schedulers, RPS Scheduler and Instance Startup Scheduler, you can simulate different types of load.
Instace Startup Scheduler controls the level of parallelism and RPS Scheduler controls throughput.

If you set RPS Scheduler to 'unlimited' and then gradually raise the number of Instances in your system by using Instance
Startup Scheduler, you'll be able to study the [scalability](http://www.perfdynamics.com/Manifesto/USLscalability.html)
of your service. 

If you set Instances count to a big, unchanged value (you can estimate the needed amount by using
[Little's Law](https://en.wikipedia.org/wiki/Little%27s_law)) and then gradually raise the RPS by using RPS Scheduler,
you'll be able to simulate Internet and push your service to its limits.

You can also combine two methods mentioned above. And, one more thing, RPS Scheduler can control a whole Instances Pool or
each Instance individually.

#### Instances and Guns
Instances takes an ammo, waits for a Scheduler tick and then shoots with a Gun it has. Gun is a tool that sends
a request to your service and measures the parameters (time, error codes, etc.) of the response.

#### Aggregator
Aggregator collects measured samples and saves them somewhere.
