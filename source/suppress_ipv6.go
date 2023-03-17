package source

import (
	"context"
	"net"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
)

type suppressIPv6Source struct {
	unfiltered Source
}

func NewSuppressIPv6Source(original Source) Source {
	return &suppressIPv6Source{
		unfiltered: original,
	}
}

func getIp4Targets(targets endpoint.Targets) endpoint.Targets {
	result := []string{}
	for _, target := range targets {
		ip := net.ParseIP(target)
		if ip != nil && ip.To4() != nil {
			// This is an IPv4
			result = append(result, target)
		} else {
			log.Debugf("Suppressed %s, not IPv4 address", target)
		}
	}
	return result
}

func (s *suppressIPv6Source) Endpoints(ctx context.Context) ([]*endpoint.Endpoint, error) {
	endpoints, err := s.unfiltered.Endpoints(ctx)
	if err != nil {
		return endpoints, err
	}
	results := []*endpoint.Endpoint{}
	for _, endpoint := range endpoints {
		targets := getIp4Targets(endpoint.Targets)
		if len(targets) > 0 {
			endpointCopy := *endpoint
			endpointCopy.Targets = targets

			results = append(results, &endpointCopy)
		} else {
			log.Debugf("Suppressed %s. No IPv4 targets", endpoint.DNSName)
		}
	}

	return results, nil
}

func (s *suppressIPv6Source) AddEventHandler(ctx context.Context, f func()) {
	s.unfiltered.AddEventHandler(ctx, f)
}
