[Home](../../index.md)

---

# RPS per instance

Usually in tests, when we increase the speed of requests submitted to the target service by specifying the `line`, `const`, `step` scheme in the rps section,
then in the `startup` section we specify the `once` scheme, because we want all instances to be available from the very beginning of the test in order to generate the load we need.

In tests with scenario load, when we have each instance describing virtual user behavior, then in the `startup` section we can specify a smooth growth of the number of users, for example, with the scheme `instance_step`, increasing their number step by step, or `const`, increasing the number of users at a constant rate.
The `rps-per-instance` instance pool setting can be used for this purpose. It is useful for the script generator when we want to limit the speed of each user in rps.

For example we specify `const` and enable `rps-per-instance`, then by increasing users via `instance_step` we simulate the real user load.


Example:

```yaml
pools:
  - id: "scenario"
    ammo:
      type: http/scenario
      file: http_payload.hcl
    result:
      type: discard
    gun:
      target: localhost:443
      type: http/scenario
      answlog:
        enabled: false
    rps-per-instance: true
    rps:
      - type: const
        duration: 1m
        ops: 4
    startup:
      type: instance_step
      from: 10
      to: 100
      step: 10
      stepduration: 10s
log:
  level: error
```

---

[Home](../../index.md)
