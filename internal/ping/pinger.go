package ping

import (
	"context"
	"fmt"

	"netprobe/internal/target"
)

// CompositePinger implements Pinger for ICMP, ARP, and NDP
// Supports both IPv4 (ICMP + ARP) and IPv6 (ICMP + NDP)
type CompositePinger struct {
	icmp *ICMPPinger
	arp  *ARPPinger
	ndp  *NDPPinger
}

// NewCompositePinger creates a pinger that handles ICMP, ARP (IPv4), and NDP (IPv6)
func NewCompositePinger(icmpTimeoutMS, icmpCount, arpTimeoutMS, ndpTimeoutMS int) *CompositePinger {
	return &CompositePinger{
		icmp: NewICMPPinger(icmpTimeoutMS, icmpCount),
		arp:  NewARPPinger(arpTimeoutMS),
		ndp:  NewNDPPinger(ndpTimeoutMS),
	}
}

// Ping routes to the appropriate pinger based on method
func (d *CompositePinger) Ping(ctx context.Context, t target.Target, method string) (PingResult, error) {
	switch method {
	case "icmp":
		return d.icmp.Ping(ctx, t, method)
	case "arp":
		return d.arp.Ping(ctx, t, method)
	case "ndp":
		return d.ndp.Ping(ctx, t, method)
	default:
		return PingResult{}, fmt.Errorf("unknown ping method: %s", method)
	}
}
