[Home](../../index.md)

---

# Discard Overflow

When you specify a [load profile](../load-profile.md), the generator calculates the order and timing of requests. This
can be referred to as the schedule of requests execution. Pandora's scheduler is responsible for this. It receives
requests from the provider and according to this schedule, passes them to the instances. Each instance then executes the
requests sequentially.

There may be situations where the scheduler believes it's time to execute the next request, but all instances are busy
waiting for their current requests to complete. In this case, the scheduler can proceed in one of two ways:

1. Wait until an instance becomes available and then pass the request to it later than scheduled.
2. Discard this request and wait for the next one, anticipating that by the time the next request is due, one of the
   instances will have become available.

The instance setting `discard_overflow` determines which behavior to follow.

1. `discard_overflow: false` - Flexible schedule adherence. The generator ensures that all planned requests are sent.
   The test duration depends on the performance of the service being tested, average response time, and the number of
   instances.
2. `discard_overflow: true` - Strict adherence to the request schedule by the generator. Requests that do not fit into
   the schedule are discarded. The test duration is predetermined. Requests that fail to meet the schedule are marked as
   failed (with a net error `777`, and also tagged as discarded). Pandora considers a test to have failed schedule, if 
   the time of the request is 2 seconds behind. That is 2 second sliding window is used.

By default, starting from version pandora@0.5.24, the setting `discard_overflow: true` is enabled.

## A Bit of Theory

When might the situation arise that forces the scheduler to choose the `discard_overflow` behavior? As mentioned
earlier, this occurs when it's time to execute a request, but there are no free instances available to process it. Why
can this happen? This typically occurs when the server's response time is high and the combination of the number of
instances and the load profile is not optimally selected. This is when `V > N * 1/rps`, where:

- `V` is the response time of the server being tested (in seconds).
- `N` is the number of instances.

To avoid such situations, you can:

- Increase the number of instances.
- Decrease the load profile.
- Optimize the server being tested to reduce response time.

---

[Home](../../index.md)
