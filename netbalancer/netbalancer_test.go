package netbalancer

import (
	"net"
	"testing"

	"github.com/CAFxX/balancer"
)

func Test_equalsHost(t *testing.T) {
	type args struct {
		a balancer.Host
		b balancer.Host
	}
	tests := []struct {
		name string
		args args
		want bool
	}{{
		name: "two equal hosts",
		args: args{
			a: balancer.Host{Address: net.IPv4(192, 168, 0, 1), Port: 123},
			b: balancer.Host{Address: net.IPv4(192, 168, 0, 1), Port: 123},
		},
		want: true,
	},
		{
			name: "two not equal hosts by port",
			args: args{
				a: balancer.Host{Address: net.IPv4(192, 168, 0, 1), Port: 1233},
				b: balancer.Host{Address: net.IPv4(192, 168, 0, 1), Port: 1234},
			},
			want: false,
		},
		{
			name: "two not equal hosts by IP",
			args: args{
				a: balancer.Host{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
				b: balancer.Host{Address: net.IPv4(192, 168, 0, 1), Port: 1234},
			},
			want: false,
		},
		{
			name: "two not equal hosts by IP and Port",
			args: args{
				a: balancer.Host{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
				b: balancer.Host{Address: net.IPv4(192, 168, 0, 1), Port: 123},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := equalsHost(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("equalsHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_equals(t *testing.T) {
	type args struct {
		a []balancer.Host
		b []balancer.Host
	}
	tests := []struct {
		name string
		args args
		want bool
	}{{
		name: "empty list",
		args: args{
			a: []balancer.Host{},
			b: []balancer.Host{},
		},
		want: true,
	}, {
		name: "two same ips",
		args: args{
			a: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
			},
			b: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
			},
		},
		want: true,
	}, {
		name: "different list length",
		args: args{
			a: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 1), Port: 1234},
			},
			b: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
			},
		},
		want: false,
	}, {
		name: "two equals lists of hosts",
		args: args{
			a: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 1), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 3), Port: 1234},
			},
			b: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 1), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 3), Port: 1234},
			},
		},
		want: true,
	}, {
		name: "two equals scrambled lists of hosts",
		args: args{
			a: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 3), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 1), Port: 1234},
			},
			b: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 1), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 2), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 3), Port: 1234},
			},
		},
		want: true,
	}, {
		name: "two equals scrambled lists of host ports",
		args: args{
			a: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 1), Port: 123},
				{Address: net.IPv4(192, 168, 0, 1), Port: 1234},
				{Address: net.IPv4(192, 168, 0, 1), Port: 12345},
			},
			b: []balancer.Host{
				{Address: net.IPv4(192, 168, 0, 1), Port: 12345},
				{Address: net.IPv4(192, 168, 0, 1), Port: 123},
				{Address: net.IPv4(192, 168, 0, 1), Port: 1234},
			},
		},
		want: true,
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := equals(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("equals() = %v, want %v", got, tt.want)
			}
		})
	}
}
