# balancer

A simple client side HTTP load balancer for go applications.

`balancer` was made to provide easier access to DNS-based load balancing for go services running in kubernetes and was mainly built for `http.Client`.

This library originated as a fork of [`github.com/esiqveland/balancer`](https://github.com/esiqveland/balancer) but has since been fully rewritten and it contains no upstream code.

## Scope

`balancer` does not do health checking and does not monitor status of any hosts.
This is left up to decide for a consul DNS or kubernetes DNS, and assumes hosts returned are deemed healthy.

`balancer` does not retry or otherwise try to fix problems, leaving this up to the caller.

`balancer` currently assumes that a lookup will return a non-empty set of initial hosts on startup.

As the  `net` and `net/http` packages do not expose the required functionality, `balancer` disables the dual stack (IPv4/IPv6) support that is normally automatically provided by those packages. Care must be taken to specify the correct stack to use. See `Wrap` for details.