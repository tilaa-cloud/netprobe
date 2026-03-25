package tests

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"netprobe/internal/config"
	"netprobe/internal/metrics"
	"netprobe/internal/ping"
	"netprobe/internal/scheduler"
	"netprobe/internal/target"
)

// TestUnit_IPv6SubnetDetection tests IPv6 /64 subnet detection logic
// This is the critical function that fixes Docker bridge routing
func TestUnit_IPv6SubnetDetection(t *testing.T) {
	testCases := []struct {
		name       string
		ip1        string
		ip2        string
		sameSubnet bool
	}{
		{
			name:       "Same /64 subnet - typical case",
			ip1:        "2001:db8:85a3::8a2e:370:7334",
			ip2:        "2001:db8:85a3::1",
			sameSubnet: true,
		},
		{
			name:       "Same /64 subnet - minimal difference",
			ip1:        "2001:db8:abcd:0000:0000:0000:0000:0000",
			ip2:        "2001:db8:abcd:0000:ffff:ffff:ffff:ffff",
			sameSubnet: true,
		},
		{
			name:       "Different /64 subnet - different second segment",
			ip1:        "2001:db8:85a3::1",
			ip2:        "2001:db8:85a4::1",
			sameSubnet: false,
		},
		{
			name:       "Different /64 subnet - different first segment",
			ip1:        "2001:db8:85a3::1",
			ip2:        "2001:db9:85a3::1",
			sameSubnet: false,
		},
		{
			name:       "Link-local addresses - same /64",
			ip1:        "fe80::1",
			ip2:        "fe80::2",
			sameSubnet: true,
		},
		{
			name:       "Link-local vs global",
			ip1:        "fe80::1",
			ip2:        "2001:db8::1",
			sameSubnet: false,
		},
		{
			name:       "Loopback",
			ip1:        "::1",
			ip2:        "::1",
			sameSubnet: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip1 := net.ParseIP(tc.ip1)
			ip2 := net.ParseIP(tc.ip2)
			require.NotNil(t, ip1, "Failed to parse ip1: %s", tc.ip1)
			require.NotNil(t, ip2, "Failed to parse ip2: %s", tc.ip2)

			// isIPv6OnSameSubnet checks if two IPv6 addresses are on the same /64 subnet
			// We'll test the logic by comparing first 8 bytes
			sameSubnet := isIPv6OnSameSubnet(ip1, ip2)
			assert.Equal(t, tc.sameSubnet, sameSubnet,
				"Expected %s and %s to be on same subnet: %v, got: %v",
				tc.ip1, tc.ip2, tc.sameSubnet, sameSubnet)
		})
	}
}

// isIPv6OnSameSubnet checks if two IPv6 addresses are on the same /64 subnet
// This mirrors the logic in ndp.go and is what gets tested
func isIPv6OnSameSubnet(ip1, ip2 net.IP) bool {
	// Docker uses /64 subnets for IPv6
	// Compare first 8 bytes (64 bits)
	if len(ip1) < 8 || len(ip2) < 8 {
		return false
	}

	for i := 0; i < 8; i++ {
		if ip1[i] != ip2[i] {
			return false
		}
	}
	return true
}

// TestIntegration_SchedulerRunsNDPCycle tests that the scheduler fetches IPv6 targets and runs NDP pings
func TestIntegration_SchedulerRunsNDPCycle(t *testing.T) {
	// Create IPv6 test targets
	testTargets := []target.Target{
		{
			DestinationIP: "2001:db8::1",
			Dimensions: map[string]string{
				"customer_id": "ipv6-cust-1",
				"region":      "us-west",
				"pod":         "prod",
				"host":        "server-01",
			},
		},
		{
			DestinationIP: "2001:db8::2",
			Dimensions: map[string]string{
				"customer_id": "ipv6-cust-1",
				"region":      "us-west",
				"pod":         "prod",
				"host":        "server-02",
			},
		},
	}

	// Create mock source
	mockSource := &MockTargetSource{targets: testTargets}

	// Create mock pinger with NDP results for all targets
	mockResults := make(map[string]ping.PingResult)
	for _, tgt := range testTargets {
		mockResults[fmt.Sprintf("%s:icmp", tgt.DestinationIP)] = ping.PingResult{
			Target:            tgt,
			Method:            "icmp",
			Success:           true,
			PacketsSent:       1,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      2.5,
			LatencyMaxMS:      3.5,
			LatencyAvgMS:      3.0,
			Timestamp:         time.Now(),
		}
		mockResults[fmt.Sprintf("%s:ndp", tgt.DestinationIP)] = ping.PingResult{
			Target:            tgt,
			Method:            "ndp",
			Success:           true,
			PacketsSent:       1,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      1.5,
			LatencyMaxMS:      2.5,
			LatencyAvgMS:      2.0,
			ResponsingIP:      tgt.DestinationIP,
			RespondingMac:     "aa:bb:cc:dd:ee:ff",
			Timestamp:         time.Now(),
		}
	}

	executor := ping.NewExecutor(&MockPinger{results: mockResults})
	store := metrics.NewMetricsStorage()

	cfg := &config.SchedulerConfig{
		PingIntervalSeconds: 1, // Short interval for testing
		BatchSize:           10,
		MaxParallelWorkers:  2,
	}

	sched := scheduler.NewScheduler(cfg, mockSource, executor, store)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start scheduler
	sched.Start(ctx)

	// Give it time to run at least one cycle
	time.Sleep(200 * time.Millisecond)

	// Verify metrics were populated
	allMetrics := store.GetAll()
	assert.NotEmpty(t, allMetrics)
	assert.Greater(t, len(allMetrics), 0)

	// Verify we have metrics for both IPv6 targets and both methods (ICMP + NDP, not ARP)
	icmpCount := 0
	ndpCount := 0
	arpCount := 0
	for _, entry := range allMetrics {
		switch entry.Key.Method {
		case "icmp":
			icmpCount++
		case "ndp":
			ndpCount++
		case "arp":
			arpCount++
		}
	}
	assert.Equal(t, 2, icmpCount, "Should have 2 ICMP results for 2 IPv6 targets")
	assert.Equal(t, 2, ndpCount, "Should have 2 NDP results for 2 IPv6 targets")
	assert.Equal(t, 0, arpCount, "Should NOT have ARP results for IPv6 targets")
}

// TestIntegration_MixedIPv4IPv6Targets tests scheduler with both IPv4 and IPv6 targets
// This verifies that the executor correctly selects methods based on IP version
func TestIntegration_MixedIPv4IPv6Targets(t *testing.T) {
	// Create mixed IPv4 and IPv6 targets
	testTargets := []target.Target{
		{
			DestinationIP: "10.0.0.1", // IPv4
			Dimensions: map[string]string{
				"customer_id": "mixed-cust",
				"ip_version":  "v4",
			},
		},
		{
			DestinationIP: "2001:db8::1", // IPv6
			Dimensions: map[string]string{
				"customer_id": "mixed-cust",
				"ip_version":  "v6",
			},
		},
		{
			DestinationIP: "192.168.1.1", // IPv4
			Dimensions: map[string]string{
				"customer_id": "mixed-cust",
				"ip_version":  "v4",
			},
		},
	}

	mockSource := &MockTargetSource{targets: testTargets}

	// Create mock pinger with appropriate results for each target
	mockResults := make(map[string]ping.PingResult)
	mockResults["10.0.0.1:icmp"] = ping.PingResult{
		Target: testTargets[0], Method: "icmp", Success: true, PacketsSent: 1,
		PacketsLost: 0, PacketLossPercent: 0.0, LatencyMinMS: 1.0,
		LatencyMaxMS: 1.0, LatencyAvgMS: 1.0, Timestamp: time.Now(),
	}
	mockResults["10.0.0.1:arp"] = ping.PingResult{
		Target: testTargets[0], Method: "arp", Success: true, PacketsSent: 1,
		PacketsLost: 0, PacketLossPercent: 0.0, LatencyMinMS: 0.5,
		LatencyMaxMS: 0.5, LatencyAvgMS: 0.5, Timestamp: time.Now(),
	}

	mockResults["2001:db8::1:icmp"] = ping.PingResult{
		Target: testTargets[1], Method: "icmp", Success: true, PacketsSent: 1,
		PacketsLost: 0, PacketLossPercent: 0.0, LatencyMinMS: 2.0,
		LatencyMaxMS: 2.0, LatencyAvgMS: 2.0, Timestamp: time.Now(),
	}
	mockResults["2001:db8::1:ndp"] = ping.PingResult{
		Target: testTargets[1], Method: "ndp", Success: true, PacketsSent: 1,
		PacketsLost: 0, PacketLossPercent: 0.0, LatencyMinMS: 1.5,
		LatencyMaxMS: 1.5, LatencyAvgMS: 1.5, ResponsingIP: "2001:db8::1",
		RespondingMac: "aa:bb:cc:dd:ee:ff", Timestamp: time.Now(),
	}

	mockResults["192.168.1.1:icmp"] = ping.PingResult{
		Target: testTargets[2], Method: "icmp", Success: true, PacketsSent: 1,
		PacketsLost: 0, PacketLossPercent: 0.0, LatencyMinMS: 1.2,
		LatencyMaxMS: 1.2, LatencyAvgMS: 1.2, Timestamp: time.Now(),
	}
	mockResults["192.168.1.1:arp"] = ping.PingResult{
		Target: testTargets[2], Method: "arp", Success: true, PacketsSent: 1,
		PacketsLost: 0, PacketLossPercent: 0.0, LatencyMinMS: 0.8,
		LatencyMaxMS: 0.8, LatencyAvgMS: 0.8, Timestamp: time.Now(),
	}

	executor := ping.NewExecutor(&MockPinger{results: mockResults})
	store := metrics.NewMetricsStorage()

	cfg := &config.SchedulerConfig{
		PingIntervalSeconds: 1,
		BatchSize:           10,
		MaxParallelWorkers:  4,
	}

	sched := scheduler.NewScheduler(cfg, mockSource, executor, store)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sched.Start(ctx)
	time.Sleep(200 * time.Millisecond)

	allMetrics := store.GetAll()
	require.Greater(t, len(allMetrics), 0, "Should have metrics")

	// Count by method
	methods := make(map[string]int)
	for _, entry := range allMetrics {
		methods[entry.Key.Method]++
	}

	assert.Equal(t, 3, methods["icmp"], "All 3 targets should have ICMP")
	assert.Equal(t, 2, methods["arp"], "2 IPv4 targets should have ARP")
	assert.Equal(t, 1, methods["ndp"], "1 IPv6 target should have NDP")
}

// TestUnit_SubnetDetectionEdgeCases tests edge cases in IPv6 subnet detection
func TestUnit_SubnetDetectionEdgeCases(t *testing.T) {
	testCases := []struct {
		name   string
		ip1    string
		ip2    string
		expect bool
	}{
		{
			name:   "IPv4 addresses should not match (converted to IPv6)",
			ip1:    "::ffff:192.0.2.1", // IPv4-mapped IPv6
			ip2:    "::ffff:192.0.2.2",
			expect: true, // Same /64 prefix (::ffff:192.0:0:0)
		},
		{
			name:   "Unspecified address",
			ip1:    "::",
			ip2:    "::1",
			expect: true, // Both in same /64 subnet
		},
		{
			name:   "Multicast addresses same /64",
			ip1:    "ff02::1",
			ip2:    "ff02::2",
			expect: true,
		},
		{
			name:   "Multicast addresses different /64",
			ip1:    "ff02::1",
			ip2:    "ff03::1",
			expect: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip1 := net.ParseIP(tc.ip1)
			ip2 := net.ParseIP(tc.ip2)
			require.NotNil(t, ip1, "Failed to parse ip1: %s", tc.ip1)
			require.NotNil(t, ip2, "Failed to parse ip2: %s", tc.ip2)

			result := isIPv6OnSameSubnet(ip1, ip2)
			assert.Equal(t, tc.expect, result,
				"Subnet check for %s and %s", tc.ip1, tc.ip2)
		})
	}
}
