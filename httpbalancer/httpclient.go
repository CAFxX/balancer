package httpbalancer

import (
	"net"
	"net/http"
	"strconv"

	"github.com/esiqveland/balancer"
	"github.com/rs/zerolog"
)

func Wrap(client *http.Client, balancer balancer.Balancer) *http.Client {
	client.Transport = NewBalancedRoundTripper(balancer, client.Transport)
	return client
}

func NewBalancedRoundTripper(balancer balancer.Balancer, delegate http.RoundTripper) http.RoundTripper {
	return &balancedRoundTripper{
		Delegate: delegate,
		Balancer: balancer,
	}
}

type balancedRoundTripper struct {
	Delegate http.RoundTripper
	Balancer balancer.Balancer
}

func (rt *balancedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	host, err := rt.Balancer.Next()
	if err != nil {
		return nil, err
	}

	req = req.Clone(req.Context())
	req.URL.Host = net.JoinHostPort(host.Address.String(), strconv.Itoa(host.Port))

	return rt.Delegate.RoundTrip(req)
}

var (
	// make sure we implement http.RoundTripper
	_ http.RoundTripper = &balancedRoundTripper{}
)
