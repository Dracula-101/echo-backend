package client

import (
	"google.golang.org/grpc/resolver"
)

type StaticResolver struct {
	target resolver.Target
	cc     resolver.ClientConn
	addrs  []resolver.Address
}

func NewStaticResolver(addrs []string) resolver.Builder {
	return &staticResolverBuilder{
		addrs: addrs,
	}
}

type staticResolverBuilder struct {
	addrs []string
}

func (b *staticResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	addresses := make([]resolver.Address, len(b.addrs))
	for i, addr := range b.addrs {
		addresses[i] = resolver.Address{Addr: addr}
	}

	r := &StaticResolver{
		target: target,
		cc:     cc,
		addrs:  addresses,
	}

	r.cc.UpdateState(resolver.State{Addresses: r.addrs})
	return r, nil
}

func (b *staticResolverBuilder) Scheme() string {
	return "static"
}

func (r *StaticResolver) ResolveNow(resolver.ResolveNowOptions) {}

func (r *StaticResolver) Close() {}
