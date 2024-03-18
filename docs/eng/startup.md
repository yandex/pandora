[Home](index.md)

---

# Instance startup profile

You can control the profile of instance starts.

This section can be thought of as how many instances you need, and how quickly they will be available to you.

Types of Instance startup profile:

- [once](#once)
- [const](#const)
- [instance_step](#instance_step)
- [composite](#composite)

#### Note: you cannot reduce the number of running instances

The startup profile only works to create instances. That is, Pandora does not delete instances until the test is complete.

## once

A specified number of instances are created once.

**Example**:

creating 10 instances at the start of this test section

```yaml
startup:
  type: once
  times: 10
```

## const

Creating instances at a certain speed.

**Example**:

creating 5 instances every second for 60 seconds. As a result, 300 instances will be created after 60 seconds

```yaml
startup:
  type: const
  duration: 60s
  ops: 5
```

## instance_step

Creates instances in periodic increments.

**Example**:

10 instances are created every 10 seconds. As a result, 100 instances will be created after 100 seconds

```yaml
startup:
  type: instance_step
  from: 10
  to: 100
  step: 10
  stepduration: 10s
```

## composite

Composite startup profile is a possibility of arbitrary combination of the above described profiles.

**Example**:

Implement a single step [instance_step](#instance_step) using once and const.
- 10 instances are created
- No instances are created within 10 seconds(_ops: 0_)
- 10 instances are created.
- As a result, 20 instances will be created and will run until the entire test is complete

```yaml
startup:
  - type: once
    times: 10
  - type: const
    ops: 0
    duration: 10s
  - type: once
    times: 10
```



---

[Home](index.md)
