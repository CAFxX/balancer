package balancer_test

import (
	"net"
	"net/http"
	"time"

	"github.com/CAFxX/balancer"
)

func Example() {
	client := http.DefaultClient
	client.Transport = balancer.Wrap(http.DefaultTransport, net.DefaultResolver, "ip4")
	// Requests using the client will now be balanced across all IPv4 addressed of the hostname.
	client.Get("http://example.com")
}

func ExampleAdvanced() {
	client := http.DefaultClient
	resolver := &balancer.CachingResolver{
		Resolver: &balancer.SingleflightResolver{
			Resolver: &balancer.TimeoutResolver{
				Resolver: net.DefaultResolver,
				Timeout:  2 * time.Second,
			},
		},
		TTL:    1 * time.Second,
		NegTTL: 250 * time.Millisecond,
	}
	client.Transport = balancer.Wrap(http.DefaultTransport, resolver, "ip4")
	// Requests using the client will now be balanced across all IPv4 addressed of the hostname,
	// using a 2 seconds timeout for the DNS requests and a DNS cache with upstream request
	// deduplication.
	client.Get("http://example.com")
}
