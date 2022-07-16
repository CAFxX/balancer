package balancer

import (
	"context"
	"net/netip"

	"resenje.org/singleflight"
)

// SingleflightResolver allows to deduplicate concurrent requests for the
// DNS records of the same hostname.
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
	addr, _, err := r.sf.Do(ctx, singleflightKey{af, host}, func(ctx context.Context) ([]netip.Addr, error) {
		defer r.sf.Forget(singleflightKey{af, host})
		return r.Resolver.LookupNetIP(ctx, af, host)
	})
	return addr, err
}
