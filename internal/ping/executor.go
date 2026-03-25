package ping

import (
	"context"

	"netprobe/internal/logger"
	"netprobe/internal/target"
)

// Pinger defines the interface for different ping methods
type Pinger interface {
	Ping(ctx context.Context, target target.Target, method string) (PingResult, error)
}

// Executor orchestrates ping operations for targets
type Executor struct {
	pinger Pinger
}

// NewExecutor creates a new ping executor
func NewExecutor(pinger Pinger) *Executor {
	return &Executor{
		pinger: pinger,
	}
}

// Ping executes appropriate ping methods on a target
// For IPv4: ICMP + ARP
// For IPv6: ICMP + NDP
func (e *Executor) Ping(ctx context.Context, target target.Target) ([]PingResult, error) {
	var results []PingResult
	logger.Debug("[EXEC] Starting ping for %s", target.DestinationIP)

	// Execute ICMP ping (works for both IPv4 and IPv6)
	logger.Debug("[EXEC] Sending ICMP ping to %s", target.DestinationIP)
	icmpResult, err := e.pinger.Ping(ctx, target, "icmp")
	if err != nil {
		logger.Debug("[EXEC] ICMP ping error: %v", err)
		icmpResult = PingResult{
			Target:            target,
			Method:            "icmp",
			Success:           false,
			PacketsSent:       1,
			PacketsLost:       1,
			PacketLossPercent: 100.0,
			Timestamp:         icmpResult.Timestamp,
		}
	}
	results = append(results, icmpResult)

	// Choose second method based on IP version
	var method string
	if IsIPv6(target.DestinationIP) {
		method = "ndp"
		logger.Debug("[EXEC] IPv6 detected, using NDP for %s", target.DestinationIP)
	} else {
		method = "arp"
		logger.Debug("[EXEC] IPv4 detected, using ARP for %s", target.DestinationIP)
	}

	// Execute ARP (IPv4) or NDP (IPv6) ping
	logger.Debug("[EXEC] Sending %s to %s", method, target.DestinationIP)
	secondResult, err := e.pinger.Ping(ctx, target, method)
	if err != nil {
		logger.Debug("[EXEC] %s error: %v", method, err)
		secondResult = PingResult{
			Target:            target,
			Method:            method,
			Success:           false,
			PacketsSent:       1,
			PacketsLost:       1,
			PacketLossPercent: 100.0,
			Timestamp:         secondResult.Timestamp,
		}
	}
	results = append(results, secondResult)

	return results, nil
}
