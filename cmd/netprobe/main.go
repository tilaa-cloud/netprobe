package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"           // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"netprobe/internal/config"
	"netprobe/internal/http"
	"netprobe/internal/logger"
	"netprobe/internal/metrics"
	"netprobe/internal/ping"
	"netprobe/internal/scheduler"
	"netprobe/internal/target"
)

// checkRawSocketPermission verifies that we have CAP_NET_RAW capability
// needed for ARP, NDP, and raw ICMP operations
func checkRawSocketPermission() bool {
	// Try to create a raw socket (requires CAP_NET_RAW)
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, syscall.ETH_P_ALL)
	if err != nil {
		return false
	}
	_ = syscall.Close(fd) // nolint: errcheck
	return true
}

func main() {
	// Initialize logger from environment
	logger.InitFromEnv()

	// Check for required capabilities
	if !checkRawSocketPermission() {
		logger.Fatal("[INIT] CAP_NET_RAW capability not available - required for ARP and NDP operations. To fix: run 'setcap cap_net_raw=ep /path/to/netprobe' or use sudo")
	}

	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration: %v", err)
	}
	logger.Info("[INIT] Configuration loaded from %s", *configPath)
	logger.Info("[INIT] Ping interval: %d seconds, batch size: %d, max workers: %d",
		cfg.SchedulerConfig.PingIntervalSeconds,
		cfg.SchedulerConfig.BatchSize,
		cfg.SchedulerConfig.MaxParallelWorkers)

	// Create metrics storage
	metricsStore := metrics.NewMetricsStorage()

	// Create Prometheus collector with configured dimensions
	collector := metrics.NewPrometheusCollector(metricsStore, cfg.Database.DimensionLabels)

	// Create pinger (ICMP, ARP for IPv4, NDP for IPv6)
	pinger := ping.NewCompositePinger(
		cfg.ICMP.TimeoutMS,
		cfg.ICMP.Count,
		cfg.ARP.TimeoutMS,
		cfg.NDP.TimeoutMS,
	)
	logger.Info("[INIT] Pinger created: ICMP timeout=%dms, count=%d, ARP timeout=%dms, NDP timeout=%dms",
		cfg.ICMP.TimeoutMS, cfg.ICMP.Count, cfg.ARP.TimeoutMS, cfg.NDP.TimeoutMS)

	// Create ping executor
	executor := ping.NewExecutor(pinger)

	// Create target source from configured database
	var targetSource target.TargetSource
	if cfg.Database.Type != "" {
		db, err := setupDatabase(&cfg.Database)
		if err != nil {
			logger.Fatal("Failed to setup database: %v", err)
		}
		logger.Info("[INIT] Database connection established: type=%s, host=%s, database=%s",
			cfg.Database.Type, cfg.Database.Host, cfg.Database.Database)
		targetSource = target.NewDatabaseSource(db, cfg.Database.Query, cfg.Database.DimensionLabels)
	} else {
		logger.Warn("[WARN] No database configured, using empty target source")
		targetSource = target.NewEmptyTargetSource()
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Exporter.ListenAddress, cfg.Exporter.ListenPort)
	httpServer := http.NewServer(addr, collector)
	logger.Info("[INIT] HTTP server configured to listen on %s", addr)

	// Start scheduler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sched := scheduler.NewScheduler(&cfg.SchedulerConfig, targetSource, executor, metricsStore)
	logger.Info("[INIT] Scheduler created and starting...")
	sched.Start(ctx)

	// Start HTTP server in a goroutine
	go func() {
		if err := httpServer.Start(); err != nil {
			logger.Error("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received, stopping")
	cancel()
	_ = httpServer.Stop() // nolint: errcheck
}

// setupDatabase creates and configures a database connection
func setupDatabase(dbConfig *config.DatabaseConfig) (*sql.DB, error) {
	var dsn string
	var driverName string

	switch dbConfig.Type {
	case "postgres", "postgresql":
		driverName = "postgres"
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.User,
			dbConfig.Password,
			dbConfig.Database,
		)
	case "mysql":
		driverName = "mysql"
		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s",
			dbConfig.User,
			dbConfig.Password,
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.Database,
		)
	case "sqlite", "sqlite3":
		driverName = "sqlite3"
		dsn = dbConfig.Host // For SQLite, "host" is the file path
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbConfig.Type)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(dbConfig.ConnMaxLifetimeSeconds) * time.Second)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
