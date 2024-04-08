[Home](../index.md)

---

# Load profile

To determine what load to create on the server, use a load profile. It sets how the load will be changed and maintained.

## const

Maintains the specified load for a certain time.

Example:

generates 10000 requests per second for 300 seconds

```yaml
rps:
    type: const
    duration: 300s
    from: 1
    ops: 10000
```

## line

Linearly increases the load in a given range over a certain period of time.

Example:

the load increases from 1 to 10000 requests per second over 180 seconds

```yaml
rps:
    type: line
    duration: 180s
    from: 1
    to: 10000
```

## step

Increases the load with the specified increment size from one value to another for a certain time.

Example:

the load increases from 10 to 100 requests per second in increments of 5 and with a step duration of 30 seconds

```yaml
rps:
    type: step
    duration: 30s
    from: 10
    to: 100
    step: 5
```

## once

Sends the specified number of requests once and completes the test. There are no restrictions on the number of requests.

Example:

sends 133 requests at the start of this test section and completes the test

```yaml
rps:
    type: once
    times: 133
```

## unlimited

Sends as many requests as the target can accept within the established connections without restrictions during the specified time.

Example:

unlimited load for 30 seconds

```yaml
rps:
    type: unlimited
    duration: 30s
```

---

[Home](../index.md)
