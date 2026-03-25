package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config represents the entire application configuration
type Config struct {
	Exporter        ExporterConfig
	Database        DatabaseConfig
	ICMP            ICMPConfig
	ARP             ARPConfig
	NDP             NDPConfig
	TargetRefresh   TargetRefreshConfig
	Logging         LoggingConfig
	SchedulerConfig SchedulerConfig
}

// ExporterConfig contains HTTP server and general exporter settings
type ExporterConfig struct {
	ListenAddress       string
	ListenPort          int
	PingIntervalSeconds int
	BatchSize           int
	MaxParallelWorkers  int
}

// DatabaseConfig contains database connection details and query configuration
type DatabaseConfig struct {
	Type                   string
	Host                   string
	Port                   int
	Database               string
	User                   string
	Password               string
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeSeconds int
	Query                  string   // SQL query to fetch targets
	DimensionLabels        []string // Dimension labels mapping (e.g., destination_ip, customer_id, vlan, pod, host)
}

// ICMPConfig contains ICMP ping settings
type ICMPConfig struct {
	Enabled   bool
	TimeoutMS int
	Count     int
}

// ARPConfig contains ARP ping settings
type ARPConfig struct {
	Enabled   bool
	TimeoutMS int
}

// NDPConfig contains NDP ping settings
type NDPConfig struct {
	Enabled   bool
	TimeoutMS int
}

// TargetRefreshConfig contains target refresh settings
type TargetRefreshConfig struct {
	IntervalSeconds int
	OnStartup       bool
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string
	Format string
}

// SchedulerConfig contains scheduler settings
type SchedulerConfig struct {
	PingIntervalSeconds int
	BatchSize           int
	MaxParallelWorkers  int
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filePath string) (*Config, error) {
	viper.SetConfigFile(filePath)
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("exporter.listen_address", "0.0.0.0")
	viper.SetDefault("exporter.listen_port", 9090)
	viper.SetDefault("exporter.ping_interval_seconds", 60)
	viper.SetDefault("exporter.batch_size", 100)
	viper.SetDefault("exporter.max_parallel_workers", 10)

	viper.SetDefault("icmp.enabled", true)
	viper.SetDefault("icmp.timeout_ms", 5000)
	viper.SetDefault("icmp.count", 1)

	viper.SetDefault("arp.enabled", true)
	viper.SetDefault("arp.timeout_ms", 5000)

	viper.SetDefault("ndp.enabled", true)
	viper.SetDefault("ndp.timeout_ms", 5000)

	viper.SetDefault("target_refresh.interval_seconds", 300)
	viper.SetDefault("target_refresh.on_startup", true)

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")

	viper.SetDefault("database.query", `SELECT destination_ip, customer_id, vlan, pod, host FROM targets WHERE active = true ORDER BY destination_ip`)
	viper.SetDefault("database.dimension_labels", []string{"destination_ip", "customer_id", "vlan", "pod", "host"})

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{
		Exporter: ExporterConfig{
			ListenAddress:       viper.GetString("exporter.listen_address"),
			ListenPort:          viper.GetInt("exporter.listen_port"),
			PingIntervalSeconds: viper.GetInt("exporter.ping_interval_seconds"),
			BatchSize:           viper.GetInt("exporter.batch_size"),
			MaxParallelWorkers:  viper.GetInt("exporter.max_parallel_workers"),
		},
		Database: DatabaseConfig{
			Type:                   viper.GetString("database.type"),
			Host:                   viper.GetString("database.host"),
			Port:                   viper.GetInt("database.port"),
			Database:               viper.GetString("database.database"),
			User:                   viper.GetString("database.user"),
			Password:               viper.GetString("database.password"),
			MaxOpenConns:           viper.GetInt("database.max_open_conns"),
			MaxIdleConns:           viper.GetInt("database.max_idle_conns"),
			ConnMaxLifetimeSeconds: viper.GetInt("database.conn_max_lifetime_seconds"),
			Query:                  viper.GetString("database.query"),
			DimensionLabels:        viper.GetStringSlice("database.dimension_labels"),
		},
		ICMP: ICMPConfig{
			Enabled:   viper.GetBool("icmp.enabled"),
			TimeoutMS: viper.GetInt("icmp.timeout_ms"),
			Count:     viper.GetInt("icmp.count"),
		},
		ARP: ARPConfig{
			Enabled:   viper.GetBool("arp.enabled"),
			TimeoutMS: viper.GetInt("arp.timeout_ms"),
		},
		NDP: NDPConfig{
			Enabled:   viper.GetBool("ndp.enabled"),
			TimeoutMS: viper.GetInt("ndp.timeout_ms"),
		},
		TargetRefresh: TargetRefreshConfig{
			IntervalSeconds: viper.GetInt("target_refresh.interval_seconds"),
			OnStartup:       viper.GetBool("target_refresh.on_startup"),
		},
		Logging: LoggingConfig{
			Level:  viper.GetString("logging.level"),
			Format: viper.GetString("logging.format"),
		},
		SchedulerConfig: SchedulerConfig{
			PingIntervalSeconds: viper.GetInt("exporter.ping_interval_seconds"),
			BatchSize:           viper.GetInt("exporter.batch_size"),
			MaxParallelWorkers:  viper.GetInt("exporter.max_parallel_workers"),
		},
	}

	return cfg, nil
}
