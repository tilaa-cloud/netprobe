package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"netprobe/internal/config"
)

// TestIntegration_ConfigLoading tests that config can be loaded from a file
func TestIntegration_ConfigLoading(t *testing.T) {
	cfg, err := config.LoadConfig("../config.example.yaml")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Greater(t, cfg.Exporter.ListenPort, 0)
	assert.Greater(t, cfg.Exporter.PingIntervalSeconds, 0)
	assert.Greater(t, cfg.Exporter.BatchSize, 0)
	assert.Greater(t, cfg.Exporter.MaxParallelWorkers, 0)
}
