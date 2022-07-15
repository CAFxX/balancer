package balancer

import (
	"context"
	"fmt"
	"math/bits"
	"math/rand"
	"net"
	"net/http"
	"net/netip"

	"github.com/CAFxX/fastrand"
)

type Resolver interface {
	LookupNetIP(ctx context.Context, af, host string) ([]netip.Addr, error)
}

// Wrap returns an http.RoundTripper wrapping the provided http.RoundTripper that
// adds DNS-based load balancing (using the provided Resolver) to requests sent by
// the HTTP client.
//
// The resolver is used to resolve the IP addresses of the hostname in each
// request. The roundtripper does not cache DNS responses, so the resolver is invoked
// for each request (you can use the CachingResolver to add caching to any Resolver).
// net.Resolver and net.DefaultResolver implement the Resolver interface.
//
// The af parameter must be one of "ip", "ip4", or "ip6", and is passed as-is to
// Resolver.LookupNetIP. If your service does not support IPv6 you should set this to
// "ip4". See net.Resolver for details.
func Wrap(rt http.RoundTripper, resolver Resolver, af string) http.RoundTripper {
	b := &balancedRoundTripper{rt: rt, resolver: resolver, af: af}
	b.rnd.Seed(rand.Uint64())
	return b
}

type balancedRoundTripper struct {
	rt       http.RoundTripper
	resolver Resolver
	af       string
	rnd      fastrand.AtomicSplitMix64
}

func (rt *balancedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Hostname()
	ips, err := rt.resolver.LookupNetIP(req.Context(), rt.af, host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("resolving hostname %q: %w", host, err)
	}

	ip := ips[0]
	if len(ips) > 1 {
		// We pick randomly to minimize the chances of thundering herds.
		ip = ips[rt.randIndex(len(ips))]
	}

	req = req.Clone(req.Context()) // RoundTrippers are not allowed to modify the original request.
	if port := req.URL.Port(); port == "" {
		req.URL.Host = ip.String()
	} else {
		req.URL.Host = net.JoinHostPort(ip.String(), port)
	}
	if req.Host == "" && ip.String() != host {
		// Since we replaced the hostname in the URL with an IP, we need to
		// add the hostname in req.Host, or otherwise servers that serve
		// multiple hostnames will not be able to know which hostname the
		// client is referring to.
		req.Host = host
	}

	return rt.rt.RoundTrip(req)
}

func (rt *balancedRoundTripper) randIndex(n int) int {
	hi, _ := bits.Mul64(rt.rnd.Uint64(), uint64(n))
	return int(hi)
}
