# balancer

[![Build status](https://github.com/CAFxX/balancer/workflows/Test/badge.svg)](https://github.com/CAFxX/balancer/actions)
[![codecov](https://codecov.io/gh/CAFxX/balancer/branch/main/graph/badge.svg)](https://codecov.io/gh/CAFxX/balancer)
[![Go Report](https://goreportcard.com/badge/github.com/CAFxX/balancer)](https://goreportcard.com/report/github.com/CAFxX/balancer) 
[![Go Reference](https://pkg.go.dev/badge/github.com/CAFxX/balancer.svg)](https://pkg.go.dev/github.com/CAFxX/balancer) :warning: API is not stable yet.

A simple client-side HTTP load balancer for go applications.

[`http.Client` does not balance requests across multiple backend servers](https://github.com/golang/go/issues/34511) (a.k.a. DNS round-robin) unless persistent connections are disabled. `balancer` was created to provide DNS-based request-level HTTP load balancing for go clients.

This library originated as a fork of [`github.com/esiqveland/balancer`](https://github.com/esiqveland/balancer) but has since been fully rewritten and it contains no upstream code.

This library also contains a number of `net.Resolver` middlewares that add useful features such as caching, timeout, and  request deduplication (for thundering herd protection).

## Scope

`balancer` does not do health checking and does not monitor status of any hosts.
This is left up to external mechanisms, and assumes hosts returned by the DNS resolver are healthy.

`balancer` does not retry or otherwise try to fix problems, leaving this up to the caller.

As the  `net` and `net/http` packages do not expose the required functionality, `balancer` disables the dual stack (IPv4/IPv6) support that is normally automatically provided by those packages. Care must be taken to specify the correct stack to use. See `Wrap` for details.
