package probes

import (
	"github.com/sid-technologies/vigil/pkg/errors"
)

// Build constructs a Probe for the given target, dispatching by Kind.
func Build(target Target) (Probe, error) {
	switch target.Kind {
	case KindICMP:
		return NewICMPProbe(target), nil
	case KindTCP:
		return NewTCPProbe(target), nil
	case KindUDPDNS:
		return NewUDPDNSProbe(target), nil
	case KindUDPSTUN:
		return NewUDPSTUNProbe(target), nil
	default:
		return nil, errors.New("unknown probe kind: %q", string(target.Kind))
	}
}
