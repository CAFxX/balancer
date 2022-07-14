package netbalancer

import (
	"bytes"
	"context"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/esiqveland/balancer"
	"github.com/pkg/errors"
)

// NewNetBalancer returns a Balancer that uses dns lookups from net.Lookup* to reload a set of hosts every updateInterval.
// We can not use TTL from dns because TTL is not exposed by the Go calls.
func New(host string, port int, updateInterval, timeout time.Duration) (balancer.Balancer, error) {
	initialHosts, err := lookupTimeout(timeout, host, port)
	if err != nil {
		return nil, err
	}
	if len(initialHosts) == 0 {
		return nil, balancer.ErrNoHosts
	}

	bal := &dnsBalancer{
		lookupAddress: host,
		port:          port,
		hosts:         initialHosts,
		interval:      updateInterval,
		quit:          make(chan struct{}, 1),
		Timeout:       timeout,
	}
	bal.rnd.Seed(rand.Uint64())

	// start update loop
	go bal.loop()

	return bal, nil
}

type dnsBalancer struct {
	lookupAddress string
	port          int
	interval      time.Duration
	quit          chan struct{}
	log           log.Logger
	Timeout       time.Duration

	rnd           fastrand.AtomicSplitMix64

	hosts         atomic.Value // []balancer.Host
}

func (b *dnsBalancer) Next() (balancer.Host, error) {
	hosts := b.hosts.Load()
	
	count := uint64(len(hosts))
	if count == 0 {
		return balancer.Host{}, balancer.ErrNoHosts
	}

	idx := b.rnd.Uint64() % count
	return *hosts[idx], nil
}

func (b *dnsBalancer) loop() {
	tick := time.NewTicker(b.interval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			b.update()
		case <-b.quit:
			return
		}
	}
}

func (b *dnsBalancer) update() {
	nextHostList, err := lookupTimeout(b.Timeout, b.lookupAddress, b.port)
	if err != nil {
		//  TODO: set hostList to empty?
		b.log.Printf("[DnsBalancers] error looking up dns='%v': %v", b.lookupAddress, err)
	} else {
		if nextHostList != nil {
			prev := b.hosts.Load()
			if !equals(prev, nextHostList) {
				b.log.Printf("[DnsBalancer] hosts changed dns=%v hosts=%v", b.lookupAddress, nextHostList)
				b.hosts.CompareAndSwap(prev, nextHostList)
			}
		}
	}
}

func equalsHost(a balancer.Host, b balancer.Host) bool {
	if a.Port != b.Port {
		return false
	}

	// dont use IP.Equal because it considers ipv4 and ipv6 address to be the same.
	return bytes.Equal(a.Address, b.Address)
}

func equals(a []balancer.Host, b []balancer.Host) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !hostListContains(b, a[i]) {
			return false
		}
	}

	return true
}

func hostListContains(hosts []balancer.Host, host *balancer.Host) bool {
	for i := range hosts {
		if equalsHost(hosts[i], host) {
			return true
		}
	}

	return false
}

func lookupTimeout(timeout time.Duration, host string, port int) ([]balancer.Host, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return lookup(ctx, host, port)
}

func lookup(ctx context.Context, host string, port int) ([]balancer.Host, error) {
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return []balancer.Host{}, errors.Wrapf(err, "Error looking up host=%v", host)
	}

	hosts := make([]balancer.Host{}, 0, len(ips))
	for _, ip := range ips {
		entry := balancer.Host{
			Address: ip.IP,
			Port:    port,
		}
		hosts = append(hosts, entry)
	}

	return hosts, nil
}

func (b *dnsBalancer) Close() error {
	// TODO: wait for exit
	b.quit <- struct{}{}

	return nil
}
