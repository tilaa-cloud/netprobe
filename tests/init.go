package tests

import (
	"os"

	"netprobe/internal/logger"
)

func init() {
	// Initialize logger from environment, defaulting to Error level for tests
	// This suppresses the verbose [CYCLE], [BATCH], [EXEC], [RESULT] logs
	// Users can set LOG_LEVEL=debug or LOG_LEVEL=trace for detailed output
	if os.Getenv("LOG_LEVEL") == "" {
		// Default to Error level for test runs (much quieter)
		// Only shows [ERROR], [WARN], and fatal messages
		logger.SetLevel(logger.LevelError)
	} else {
		// User specified a level, use it
		logger.InitFromEnv()
	}
}
