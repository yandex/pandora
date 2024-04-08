[Home](../../index.md)

---

# Use shared client

## General principle

### Transport client

By default, Pandora components automatically assign it a transport client, such as for http, grpc, or tcp, when creating 
a new Instance. When the Instance starts, it opens a connection, such as a tcp connection. 
Normally, clients can use multiple connections at the same time, but in the case of Pandora, 
each Instance opens only one connection as the Instance makes requests one after the other.

It's interesting to note that creating a connection doesn't mean that requests will go through that connection. Why? 
In the test configuration, you can specify a large number of instances and a small number of RPSs. 
The Pandora provider generates requests with the frequency specified in the RPS settings 
and sends them to a random instance so that the instance will execute the request.

## `shared-client`.
In the [http](../http-generator.md) and [grpc](../grpc-generator.md) generator settings,
you can specify the `shared-client.enabled=true` parameter. If you enable this setting,
all instances will use a shared transport client and each will not have to create its own.

## `shared-client.client-number`.

The transport client uses to connect the connection. For example, HTTP and gRPC use a tcp connection.
A single client uses multiple connections and can create additional connections if needed.

But under heavy loads there may be a situation when the client does not have time to create connections.
You can increase the speed of connection creation by a common client by increasing the `shared-client.client-number` parameter.
By default `shared-client.client-number=1`.

---

[Home](../../index.md)
