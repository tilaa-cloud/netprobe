package scheduler

import (
	"context"
	"math"
	"time"

	"netprobe/internal/config"
	"netprobe/internal/logger"
	"netprobe/internal/metrics"
	"netprobe/internal/ping"
	"netprobe/internal/target"
)

// Scheduler orchestrates the ping cycle
type Scheduler struct {
	config       *config.SchedulerConfig
	targetSource target.TargetSource
	executor     *ping.Executor
	storage      *metrics.MetricsStorage
}

// NewScheduler creates a new scheduler
func NewScheduler(
	cfg *config.SchedulerConfig,
	targetSource target.TargetSource,
	executor *ping.Executor,
	storage *metrics.MetricsStorage,
) *Scheduler {
	return &Scheduler{
		config:       cfg,
		targetSource: targetSource,
		executor:     executor,
		storage:      storage,
	}
}

// Start begins the scheduler's ping cycle loop
func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
}

// run is the main scheduler loop
func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(s.config.PingIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Run once immediately
	s.executePingCycle(ctx)

	for {
		select {
		case <-ticker.C:
			s.executePingCycle(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// executePingCycle fetches targets, divides into batches, and pings them
func (s *Scheduler) executePingCycle(ctx context.Context) {
	cycleStartTime := time.Now()
	logger.Debug("[CYCLE] Started ping cycle")

	// Fetch targets from source
	targets, err := s.targetSource.Fetch(ctx)
	if err != nil {
		logger.Error("[ERROR] Failed to fetch targets: %v", err)
		return
	}
	logger.Debug("[CYCLE] Fetched %d targets from source", len(targets))

	if len(targets) == 0 {
		logger.Debug("[CYCLE] No targets to ping")
		return
	}

	// Create worker pool
	pool := ping.NewWorkerPool(s.config.MaxParallelWorkers, s.executor)

	// Start collecting results in a goroutine to prevent deadlock
	resultsDone := make(chan struct{})
	go func() {
		s.collectResults(pool)
		close(resultsDone)
	}()

	// Process targets in batches
	s.processBatches(ctx, pool, targets)

	// Signal no more jobs coming and wait for workers to finish
	pool.CloseJobs()
	pool.WaitForWorkers()
	pool.CloseResults()

	// Wait for results collection to complete
	<-resultsDone

	elapsed := time.Since(cycleStartTime)
	logger.Debug("[CYCLE] Completed in %v ms", elapsed.Milliseconds())
}

// processBatches divides targets into batches and submits them sequentially
func (s *Scheduler) processBatches(ctx context.Context, pool *ping.WorkerPool, targets []target.Target) {
	batchSize := s.config.BatchSize
	numBatches := (len(targets) + batchSize - 1) / batchSize
	logger.Debug("[BATCH] Processing %d targets in %d batches (size=%d)", len(targets), numBatches, batchSize)

	for batchNum := 0; batchNum < numBatches; batchNum++ {
		i := batchNum * batchSize
		end := i + batchSize
		if end > len(targets) {
			end = len(targets)
		}

		batch := targets[i:end]
		logger.Debug("[BATCH] Submitting batch %d with %d targets", batchNum+1, len(batch))
		pool.SubmitBatch(batch)

		// Brief pause between batches to avoid overwhelming the system
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

// collectResults reads ping results from the worker pool and updates metrics
func (s *Scheduler) collectResults(pool *ping.WorkerPool) {
	resultsChannel := pool.CollectResults()
	resultCount := 0

	// Collect all results until channel is closed
	for result := range resultsChannel {
		resultCount++
		if result.Success {
			logger.Debug("[RESULT] %s via %s (IP=%s, MAC=%s, latency=%.2fms, loss=%.1f%%)",
				result.Target.DestinationIP,
				result.Method,
				result.ResponsingIP,
				result.RespondingMac,
				result.LatencyAvgMS,
				result.PacketLossPercent)
		} else {
			logger.Debug("[RESULT] %s via %s FAILED (loss=100%%)", result.Target.DestinationIP, result.Method)
		}
		// Convert result to metric and store
		// Copy dimensions from target (which now contains all dimensions from database)
		dimensions := make(map[string]string)
		for k, v := range result.Target.Dimensions {
			dimensions[k] = v
		}

		key := metrics.MetricKey{
			DestinationIP: result.Target.DestinationIP,
			Method:        result.Method,
			ResponsingIP:  result.ResponsingIP,
			RespondingMac: result.RespondingMac,
			Dimensions:    dimensions,
		}

		// For failed pings (100% loss), set latency to NaN to indicate no value
		latencyMin := result.LatencyMinMS
		latencyMax := result.LatencyMaxMS
		latencyAvg := result.LatencyAvgMS
		if result.PacketLossPercent == 100.0 {
			latencyMin = math.NaN()
			latencyMax = math.NaN()
			latencyAvg = math.NaN()
		}

		value := metrics.MetricValue{
			PacketLossPercent: result.PacketLossPercent,
			LatencyMinMS:      latencyMin,
			LatencyMaxMS:      latencyMax,
			LatencyAvgMS:      latencyAvg,
			Timestamp:         result.Timestamp,
		}

		s.storage.Update(key, value)
	}
	logger.Debug("[CYCLE] Collected and stored %d metric results", resultCount)
}
