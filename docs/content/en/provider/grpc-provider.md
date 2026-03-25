---
title: gRPC Ammo providers
description: gRPC Ammo provider is a source of test data - it makes ammo object
categories: [Provider]
tags: [provider, grpc]
weight: 10
---

gRPC Ammo provider is a source of test data: it makes ammo object.

There is a common rule for any (built-in) provider: data supplied by ammo provider are records that will be pushed via established connection to external host (defined in pandora config via _pool.gun.target_ option). Thus, you cannot define in the ammofile to which _physical_ host your ammo will be sent.

## Test data

### grpc/json

jsonline format, 1 row — 1 json-encoded ammo.

Ammofile sample:

```json
{"tag": "/Add", "call": "api.Adder.Add", "metadata": {"Authorization": "Bearer $YC_TOKEN"}, "payload": {"x": 21, "y": 12}}
{"tag": "/Add", "call": "api.Adder.Add", "metadata": {"Authorization": "Bearer $YC_TOKEN"}, "payload": {"x": 22, "y": 13}}
{"tag": "/Add", "call": "api.Adder.Add", "metadata": {"Authorization": "Bearer $YC_TOKEN"}, "payload": {"x": 23, "y": 14}}
```

Where:
   * `tag`: Request label used in the result display interface. It enables using different tags for requests to group and filter test results.
   * `call`: Full name of the service and its called method. The service and method can be separated by `.` or `/`. For example, `api.Adder.Add`, `api.Adder/Add` or `/api.Adder/Add`.
   * `metadata`: Used to send headers, e.g., `Authorization`.
   * `payload`: Request body.

When configuring the Pandora load generator using a YAML file, specify the `grpc/json` type in the `ammo` section. Config sample:

```yaml
pools:
  - ammo:
      type: grpc/json     # ammo format
      file: ./ammo.json   # ammo file path
```

If the lines in a ammo file are more than 64 KB in length, specify the buffer size for reading lines using the `maxammosize` parameter. The value is specified in bytes and must be greater than the longest string. Configuration example:

```yaml
pools:
  - ammo:
      type: grpc/json       # ammo format
      file: ./ammo.json     # ammo file path
      maxammosize: 1000000  # maximum ammo size
```

## Features

### Ammo filters

Each http ammo provider lets you choose specific ammo for your test from ammo file with `chosencases` setting:

```yaml
pools:
  - ammo:
      type: grpc/json                  # ammo format
      chosencases: [ "tag1", "tag2" ]  # use only "tag1" and "tag2" ammo for this test
      file: ./ammofile                 # ammo file path
```

### Limiting the number of ammos

By default, all providers start reading from the beginning when they reach the end of the ammos.
The `passes` parameter is used to limit the number of repetitions.

If you need to set a general limit on the number of ammos that a provider can issue, you should use the `limit` parameter.

Example config:

```yaml
pools:
  - ammo:
      type: grpc/json     # ammo format
      file: ./ammo.json   # ammo file path
      limit: 0            # Limit limits total num of ammo. Unlimited if zero. Default is 0
      passes: 0           # Passes limits ammo file passes. Unlimited if zero. Default is 0
```
