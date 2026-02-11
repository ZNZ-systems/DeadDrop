package domain

import "net"

// DNSResolver abstracts DNS lookups so they can be replaced in tests.
type DNSResolver interface {
	LookupTXT(host string) ([]string, error)
}

// NetResolver implements DNSResolver using the standard library.
type NetResolver struct{}

func (r *NetResolver) LookupTXT(host string) ([]string, error) {
	return net.LookupTXT(host)
}
