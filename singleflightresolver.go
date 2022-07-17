package balancer

import (
	"context"
	"net/netip"

	"resenje.org/singleflight"
)

// SingleflightResolver allows to deduplicate concurrent requests for the
// DNS records of the same hostname.
//
// If multiple concurrent requests to resolve the same hostname are performed,
// SingleflightResolver will allow the first to proceed. Additional requests
// are paused until the parent resolver has responded, at which point the
// response is provided to all pending requests for that hostname.
// Note that the request to the parent resolver is cancelled only once all
// pending requests for that hostname have been cancelled.
type SingleflightResolver struct {
	Resolver Resolver // Wrapped DNS resolver.

	sf singleflight.Group[singleflightKey, []netip.Addr]
}

type singleflightKey struct {
	af   string
	host string
}

var _ Resolver = &SingleflightResolver{}

func (r *SingleflightResolver) LookupNetIP(ctx context.Context, af, host string) ([]netip.Addr, error) {
	addr, shared, err := r.sf.Do(ctx, singleflightKey{af, host}, func(ctx context.Context) ([]netip.Addr, error) {
		defer r.sf.Forget(singleflightKey{af, host})
		return r.Resolver.LookupNetIP(ctx, af, host)
	})
	if shared {
		addr = clone(addr)
	}
	return addr, err
}
