package domain

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// DNSResolver abstracts DNS lookups so they can be replaced in tests.
type DNSResolver interface {
	LookupTXT(host string) ([]string, error)
}

// NetResolver implements DNSResolver using the standard library.
type NetResolver struct{}

func (r *NetResolver) LookupTXT(host string) ([]string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, errors.New("host is required")
	}

	// Query system resolver first, then fallback to well-known public resolvers.
	// Some hosts cache stale TXT values for longer than expected.
	sources := []func(string) ([]string, error){
		func(h string) ([]string, error) { return net.LookupTXT(h) },
		func(h string) ([]string, error) { return lookupTXTWithResolver(h, "1.1.1.1:53") },
		func(h string) ([]string, error) { return lookupTXTWithResolver(h, "8.8.8.8:53") },
	}

	seen := make(map[string]struct{})
	results := make([]string, 0, 4)
	var errs []string

	for _, source := range sources {
		records, err := source(host)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		for _, record := range records {
			record = strings.TrimSpace(record)
			if record == "" {
				continue
			}
			if _, exists := seen[record]; exists {
				continue
			}
			seen[record] = struct{}{}
			results = append(results, record)
		}
	}

	if len(results) > 0 {
		return results, nil
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("txt lookup failed: %s", strings.Join(errs, "; "))
	}

	return nil, nil
}

func lookupTXTWithResolver(host, resolverAddr string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 2 * time.Second}
			return d.DialContext(ctx, "udp", resolverAddr)
		},
	}

	return resolver.LookupTXT(ctx, host)
}
