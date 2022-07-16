package balancer

import (
	"context"
	"net/netip"
	"time"
)

// TimeoutResolver allows to specify a timeout for DNS requests.
type TimeoutResolver struct {
	Resolver Resolver      // Wrapped DNS resolver.
	Timeout  time.Duration // How long to wait for a response from the wrapped DNS resolver. 0 disables the timeout.
}

var _ Resolver = &TimeoutResolver{}

func (t *TimeoutResolver) LookupNetIP(ctx context.Context, af, host string) ([]netip.Addr, error) {
	if t.Timeout > 0 {
		tctx, cancel := context.WithTimeout(ctx, t.Timeout)
		defer cancel()
		ctx = tctx
	}
	return t.Resolver.LookupNetIP(ctx, af, host)
}
