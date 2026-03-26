package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"netprobe/internal/config"
	"netprobe/internal/metrics"
	"netprobe/internal/ping"
	"netprobe/internal/scheduler"
	"netprobe/internal/target"
)

// TestIntegration_PacketLossTracking tests that packet loss is properly tracked in metrics
func TestIntegration_PacketLossTracking(t *testing.T) {
	testTarget := target.Target{
		DestinationIP: "10.0.0.1",
		Dimensions: map[string]string{
			"customer_id": "loss-test",
			"vlan":        "test",
			"pod":         "test",
			"host":        "test-host",
		},
	}

	// Create mock pinger with packet loss
	mockResults := map[string]ping.PingResult{
		"10.0.0.1:icmp": {
			Target:            testTarget,
			Method:            "icmp",
			Success:           true,
			PacketsSent:       10,
			PacketsLost:       3,
			PacketLossPercent: 30.0, // 3 out of 10 packets lost
			LatencyMinMS:      1.0,
			LatencyMaxMS:      5.0,
			LatencyAvgMS:      3.0,
			Timestamp:         time.Now(),
		},
		"10.0.0.1:arp": {
			Target:            testTarget,
			Method:            "arp",
			Success:           true,
			PacketsSent:       10,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      0.5,
			LatencyMaxMS:      2.0,
			LatencyAvgMS:      1.2,
			Timestamp:         time.Now(),
		},
	}

	executor := ping.NewExecutor(&MockPinger{results: mockResults})
	store := metrics.NewMetricsStorage()

	cfg := &config.SchedulerConfig{
		PingIntervalSeconds: 1,
		BatchSize:           10,
		MaxParallelWorkers:  2,
	}

	mockSource := &MockTargetSource{targets: []target.Target{testTarget}}
	sched := scheduler.NewScheduler(cfg, mockSource, executor, store)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sched.Start(ctx)
	time.Sleep(200 * time.Millisecond)

	allMetrics := store.GetAll()

	// Find ICMP and ARP metrics
	var icmpMetric, arpMetric metrics.MetricValue
	for _, entry := range allMetrics {
		switch entry.Key.Method {
		case "icmp":
			icmpMetric = entry.Value
		case "arp":
			arpMetric = entry.Value
		}
	}

	// Verify packet loss is correctly stored
	assert.Equal(t, 30.0, icmpMetric.PacketLossPercent)
	assert.Equal(t, 0.0, arpMetric.PacketLossPercent)
}

// TestIntegration_LatencyMetrics tests that min/max/avg latency is properly tracked
func TestIntegration_LatencyMetrics(t *testing.T) {
	testTarget := target.Target{
		DestinationIP: "10.0.0.1",
		Dimensions: map[string]string{
			"customer_id": "latency-test",
			"vlan":        "test",
			"pod":         "test",
			"host":        "test-host",
		},
	}

	mockResults := map[string]ping.PingResult{
		"10.0.0.1:icmp": {
			Target:            testTarget,
			Method:            "icmp",
			Success:           true,
			PacketsSent:       1,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      1.2,
			LatencyMaxMS:      4.8,
			LatencyAvgMS:      3.0,
			Timestamp:         time.Now(),
		},
	}

	executor := ping.NewExecutor(&MockPinger{results: mockResults})
	store := metrics.NewMetricsStorage()

	cfg := &config.SchedulerConfig{
		PingIntervalSeconds: 1,
		BatchSize:           10,
		MaxParallelWorkers:  2,
	}

	mockSource := &MockTargetSource{targets: []target.Target{testTarget}}
	sched := scheduler.NewScheduler(cfg, mockSource, executor, store)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sched.Start(ctx)
	time.Sleep(200 * time.Millisecond)

	allMetrics := store.GetAll()
	assert.NotEmpty(t, allMetrics)

	for _, entry := range allMetrics {
		if entry.Key.Method == "icmp" {
			assert.Equal(t, 1.2, entry.Value.LatencyMinMS)
			assert.Equal(t, 4.8, entry.Value.LatencyMaxMS)
			assert.Equal(t, 3.0, entry.Value.LatencyAvgMS)
		}
	}
}
