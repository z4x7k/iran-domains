package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"
)

type ResolveOption struct {
	retries int
}

type ResolveOptionFunc func(*ResolveOption)

func WithRetries(count int) ResolveOptionFunc {
	return func(opt *ResolveOption) {
		opt.retries = count
	}
}

func IsDomainResolvable(ctx context.Context, domain string, opts ...ResolveOptionFunc) (bool, error) {
	var option ResolveOption
	for _, fn := range opts {
		fn(&option)
	}

	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, network, "8.8.8.8:53")
		},
	}
	ips, err := r.LookupHost(ctx, domain)
	if nil != err {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			if option.retries > 0 {
				option.retries--
				return IsDomainResolvable(ctx, domain, opts...)
			}
		}
		return false, fmt.Errorf("failed to lookup domain: %v", err)
	}

	for _, v := range ips {
		addr, err := netip.ParseAddr(v)
		if nil != err {
			return false, fmt.Errorf("resolved ip address is unparsable: %s", v)
		}
		if !addr.IsValid() || addr.IsPrivate() || addr.IsUnspecified() || addr.IsMulticast() || addr.IsLoopback() {
			return false, fmt.Errorf("resolved ip is not a valid public unicast ip address: %s", v)
		}
	}
	return true, nil
}
