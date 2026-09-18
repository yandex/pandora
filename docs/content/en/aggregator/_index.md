---
title: Aggregator
description: 
categories: [Aggregator]
weight: 10
---

The aggregator collects samples and stores them somewhere.

### 1. phout

**All config**

```yaml
result:
  type: phout
  destination: file_path.log
  id: false # Print ammo ids if true.
  flush-time: 1s    # Buffer flush period, a string with a unit: from 1ms to 1m.
  sample-queue-size: 262144
  buffer-size: 1048576
```

### 2. jsonlines

```yaml
result:
  type: jsonlines
  sink: 
    type: file
    path: file_path
  buffer-size: 1048576
  flush-interval: 1s
  sample-queue-size: 131072
  marshal-float-with-6-digits: false
  sort-map-keys: false
```

See [here](./sink.md) for other types for `sink`.

### 3. json

This is an alias for `jsonlines`.

### 4. log

Output aggregator data to Pandora's standard logs

```yaml
result:
  type: log
```

### 5. discard

Discard aggregator output

```yaml
result:
  type: discard
```
