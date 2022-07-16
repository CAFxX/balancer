# balancer

A simple client side HTTP load balancer for go applications.

[`http.Client` does not balance requests across multiple backend servers](https://github.com/golang/go/issues/34511) (a.k.a. DNS round-robin) unless persistent connections are disabled. `balancer` was created to provide DNS-based load balancing for go services.

This library originated as a fork of [`github.com/esiqveland/balancer`](https://github.com/esiqveland/balancer) but has since been fully rewritten and it contains no upstream code.

## Scope

`balancer` does not do health checking and does not monitor status of any hosts.
This is left up to decide for a consul DNS or kubernetes DNS, and assumes hosts returned are deemed healthy.

`balancer` does not retry or otherwise try to fix problems, leaving this up to the caller.

As the  `net` and `net/http` packages do not expose the required functionality, `balancer` disables the dual stack (IPv4/IPv6) support that is normally automatically provided by those packages. Care must be taken to specify the correct stack to use. See `Wrap` for details.
