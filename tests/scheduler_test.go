package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"netprobe/internal/config"
	"netprobe/internal/metrics"
	"netprobe/internal/ping"
	"netprobe/internal/scheduler"
	"netprobe/internal/target"
)

// TestIntegration_SchedulerRunsPingCycle tests that the scheduler fetches targets and runs ping cycles
func TestIntegration_SchedulerRunsPingCycle(t *testing.T) {
	// Create test targets
	testTargets := []target.Target{
		{
			DestinationIP: "10.0.0.1",
			Dimensions: map[string]string{
				"customer_id": "cust1",
				"vlan":        "prod",
				"pod":         "us-west-2",
				"host":        "server-01",
			},
		},
		{
			DestinationIP: "10.0.0.2",
			Dimensions: map[string]string{
				"customer_id": "cust1",
				"vlan":        "prod",
				"pod":         "us-west-2",
				"host":        "server-02",
			},
		},
	}

	// Create mock source
	mockSource := &MockTargetSource{targets: testTargets}

	// Create mock pinger with results for all targets
	mockResults := make(map[string]ping.PingResult)
	for _, tgt := range testTargets {
		mockResults[fmt.Sprintf("%s:icmp", tgt.DestinationIP)] = ping.PingResult{
			Target:            tgt,
			Method:            "icmp",
			Success:           true,
			PacketsSent:       1,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      1.0,
			LatencyMaxMS:      3.0,
			LatencyAvgMS:      2.0,
			Timestamp:         time.Now(),
		}
		mockResults[fmt.Sprintf("%s:arp", tgt.DestinationIP)] = ping.PingResult{
			Target:            tgt,
			Method:            "arp",
			Success:           true,
			PacketsSent:       1,
			PacketsLost:       0,
			PacketLossPercent: 0.0,
			LatencyMinMS:      0.5,
			LatencyMaxMS:      2.5,
			LatencyAvgMS:      1.5,
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

	// Give it time to run at least one cycle (mocked pings return instantly)
	time.Sleep(200 * time.Millisecond)

	// Verify metrics were populated
	allMetrics := store.GetAll()
	assert.NotEmpty(t, allMetrics)
	assert.Greater(t, len(allMetrics), 0)

	// Verify we have metrics for both targets and both methods
	icmpCount := 0
	arpCount := 0
	for _, entry := range allMetrics {
		switch entry.Key.Method {
		case "icmp":
			icmpCount++
		case "arp":
			arpCount++
		}
	}
	assert.Equal(t, 2, icmpCount) // 2 targets with ICMP
	assert.Equal(t, 2, arpCount)  // 2 targets with ARP
}

// TestIntegration_BatchProcessing tests that targets are processed in batches
func TestIntegration_BatchProcessing(t *testing.T) {
	// Create 25 test targets
	testTargets := make([]target.Target, 25)
	for i := 0; i < 25; i++ {
		testTargets[i] = target.Target{
			DestinationIP: fmt.Sprintf("10.0.0.%d", i+1),
			Dimensions: map[string]string{
				"customer_id": "batch-test",
				"vlan":        "test",
				"pod":         "test",
				"host":        fmt.Sprintf("host-%d", i),
			},
		}
	}

	mockSource := &MockTargetSource{targets: testTargets}

	// Create mock pinger
	mockResults := make(map[string]ping.PingResult)
	for _, tgt := range testTargets {
		for _, method := range []string{"icmp", "arp"} {
			key := fmt.Sprintf("%s:%s", tgt.DestinationIP, method)
			mockResults[key] = ping.PingResult{
				Target:            tgt,
				Method:            method,
				Success:           true,
				PacketsSent:       1,
				PacketsLost:       0,
				PacketLossPercent: 0.0,
				LatencyMinMS:      1.0,
				LatencyMaxMS:      2.0,
				LatencyAvgMS:      1.5,
				Timestamp:         time.Now(),
			}
		}
	}

	executor := ping.NewExecutor(&MockPinger{results: mockResults})
	store := metrics.NewMetricsStorage()

	cfg := &config.SchedulerConfig{
		PingIntervalSeconds: 1,
		BatchSize:           10, // Process 10 targets per batch
		MaxParallelWorkers:  2,  // 2 parallel workers
	}

	sched := scheduler.NewScheduler(cfg, mockSource, executor, store)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sched.Start(ctx)
	time.Sleep(600 * time.Millisecond) // Need time for 3 batches with 100ms delays between them

	// Verify all targets were processed
	allMetrics := store.GetAll()
	// 25 targets * 2 methods = 50 metric entries
	assert.Equal(t, 50, len(allMetrics))
}
