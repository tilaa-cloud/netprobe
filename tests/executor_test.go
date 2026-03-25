package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"netprobe/internal/ping"
	"netprobe/internal/target"
)

// TestIntegration_PingExecutor tests that the ping executor runs both ICMP and ARP on targets
func TestIntegration_PingExecutor(t *testing.T) {
	testTarget := target.Target{
		DestinationIP: "10.0.0.1",
		Dimensions: map[string]string{
			"customer_id": "test-cust",
			"vlan":        "test-vlan",
			"pod":         "test-pod",
			"host":        "test-host",
		},
	}

	// Create mock pinger
	mockResults := map[string]ping.PingResult{
		"10.0.0.1:icmp": {
			Target:            testTarget,
			Method:            "icmp",
			Success:           true,
			PacketsSent:       1,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      2.0,
			LatencyMaxMS:      2.0,
			LatencyAvgMS:      2.0,
			Timestamp:         time.Now(),
		},
		"10.0.0.1:arp": {
			Target:            testTarget,
			Method:            "arp",
			Success:           true,
			PacketsSent:       1,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      1.0,
			LatencyMaxMS:      1.0,
			LatencyAvgMS:      1.0,
			Timestamp:         time.Now(),
		},
	}

	executor := ping.NewExecutor(&MockPinger{results: mockResults})
	ctx := context.Background()

	results, err := executor.Ping(ctx, testTarget)
	require.NoError(t, err)
	assert.Len(t, results, 2) // Should have results for both ICMP and ARP

	// Verify both methods were executed
	methods := make(map[string]bool)
	for _, result := range results {
		methods[result.Method] = true
		assert.True(t, result.Success)
		assert.Equal(t, 0.0, result.PacketLossPercent)
	}
	assert.True(t, methods["icmp"])
	assert.True(t, methods["arp"])
}

// TestIntegration_PingExecutor_IPv6 tests that the ping executor runs ICMP and NDP on IPv6 targets
func TestIntegration_PingExecutor_IPv6(t *testing.T) {
	testTarget := target.Target{
		DestinationIP: "2001:db8::1",
		Dimensions: map[string]string{
			"customer_id": "test-cust",
			"vlan":        "test-vlan",
			"pod":         "test-pod",
			"host":        "test-host",
		},
	}

	// Create mock pinger with results for IPv6
	mockResults := map[string]ping.PingResult{
		"2001:db8::1:icmp": {
			Target:            testTarget,
			Method:            "icmp",
			Success:           true,
			PacketsSent:       1,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      3.0,
			LatencyMaxMS:      3.0,
			LatencyAvgMS:      3.0,
			Timestamp:         time.Now(),
		},
		"2001:db8::1:ndp": {
			Target:            testTarget,
			Method:            "ndp",
			Success:           true,
			PacketsSent:       1,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      2.0,
			LatencyMaxMS:      2.0,
			LatencyAvgMS:      2.0,
			ResponsingIP:      "2001:db8::1",
			RespondingMac:     "aa:bb:cc:dd:ee:ff",
			Timestamp:         time.Now(),
		},
	}

	executor := ping.NewExecutor(&MockPinger{results: mockResults})
	ctx := context.Background()

	results, err := executor.Ping(ctx, testTarget)
	require.NoError(t, err)
	assert.Len(t, results, 2) // Should have results for both ICMP and NDP (NOT ARP)

	// Verify correct methods were executed for IPv6
	methods := make(map[string]bool)
	for _, result := range results {
		methods[result.Method] = true
		assert.True(t, result.Success)
		assert.Equal(t, 0.0, result.PacketLossPercent)
	}
	assert.True(t, methods["icmp"], "IPv6 targets should use ICMP")
	assert.True(t, methods["ndp"], "IPv6 targets should use NDP")
	assert.False(t, methods["arp"], "IPv6 targets should NOT use ARP")
}
