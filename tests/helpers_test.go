package tests

import (
	"context"
	"fmt"
	"time"

	"netprobe/internal/ping"
	"netprobe/internal/target"
)

// MockTargetSource implements target.TargetSource for testing
type MockTargetSource struct {
	targets []target.Target
}

func (m *MockTargetSource) Fetch(ctx context.Context) ([]target.Target, error) {
	return m.targets, nil
}

// MockPinger allows us to inject predetermined ping results
type MockPinger struct {
	results map[string]ping.PingResult
}

func (mp *MockPinger) Ping(ctx context.Context, target target.Target, method string) (ping.PingResult, error) {
	key := fmt.Sprintf("%s:%s", target.DestinationIP, method)
	if result, ok := mp.results[key]; ok {
		return result, nil
	}
	// Default: successful ping with no packet loss
	return ping.PingResult{
		Target:            target,
		Method:            method,
		Success:           true,
		PacketsSent:       1,
		PacketsLost:       0,
		PacketLossPercent: 0.0,
		LatencyMinMS:      1.0,
		LatencyMaxMS:      1.0,
		LatencyAvgMS:      1.0,
		Timestamp:         time.Now(),
	}, nil
}

// Helper to create IPv6 test targets
func NewIPv6Target(ip string, dimensions map[string]string) target.Target {
	return target.Target{
		DestinationIP: ip,
		Dimensions:    dimensions,
	}
}

// Helper to create IPv4 test targets
func NewIPv4Target(ip string, dimensions map[string]string) target.Target {
	return target.Target{
		DestinationIP: ip,
		Dimensions:    dimensions,
	}
}

// IPv6SubnetTest represents a test case for subnet detection
type IPv6SubnetTest struct {
	Name        string
	IP1         string
	IP2         string
	OnSameSub64 bool
}
