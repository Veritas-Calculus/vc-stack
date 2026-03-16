// Package appconfig provides centralized configuration loading for VC Stack.
//
// It uses Viper to support configuration from YAML files, environment variables,
// and defaults — in that order of precedence (env vars override YAML, which overrides defaults).
//
// Usage:
//
//	cfg, err := appconfig.Load("vc-management")
//	fmt.Println(cfg.Server.Port)       // from YAML or env VC_SERVER_PORT
//	fmt.Println(cfg.Database.Host)     // from YAML or env DB_HOST
package appconfig

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/Veritas-Calculus/vc-stack/pkg/dlock"
	"github.com/Veritas-Calculus/vc-stack/pkg/mq"
	"github.com/Veritas-Calculus/vc-stack/pkg/vcredis"
)

// AppConfig is the root configuration for vc-management.
type AppConfig struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Identity  IdentityConfig  `mapstructure:"identity"`
	Network   NetworkConfig   `mapstructure:"network"`
	Security  SecurityConfig  `mapstructure:"security"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Modules   ModulesConfig   `mapstructure:"modules"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Gateway   GatewayConfig   `mapstructure:"gateway"`
	Etcd      dlock.Config    `mapstructure:"etcd"`
	Redis     vcredis.Config  `mapstructure:"redis"`
	Kafka     mq.KafkaConfig  `mapstructure:"kafka"`

	// SDN configuration (also via env vars).
	SDNProvider    string `mapstructure:"sdn_provider"`
	BridgeMappings string `mapstructure:"bridge_mappings"`
	ExternalURL    string `mapstructure:"external_url"`

	// Ceph RGW configuration.
	CephRGWEndpoint  string `mapstructure:"ceph_rgw_endpoint"`
	CephRGWAccessKey string `mapstructure:"ceph_rgw_access_key"`
	CephRGWSecretKey string `mapstructure:"ceph_rgw_secret_key"`

	// Sentry integration.
	SentryDSN         string `mapstructure:"sentry_dsn"`
	SentryEnvironment string `mapstructure:"sentry_environment"`

	// Management TLS.
	ManagementTLS bool `mapstructure:"management_tls"`

	// Web console directory.
	WebConsoleDir string `mapstructure:"web_console_dir"`
}

// ServerConfig holds the HTTP server settings.
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	GinMode      string        `mapstructure:"gin_mode"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"` // #nosec G101 -- config field
	SSLMode         string        `mapstructure:"sslmode"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.Username, d.Password, d.Name, d.SSLMode)
}

// IdentityConfig holds JWT and auth settings.
type IdentityConfig struct {
	JWTSecret          string        `mapstructure:"jwt_secret"` // #nosec G101 -- config field
	AccessTokenExpiry  time.Duration `mapstructure:"access_token_expires_in"`
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expires_in"`
}

// NetworkConfig holds SDN and OVN settings.
type NetworkConfig struct {
	OVNNBAddress string `mapstructure:"ovn_nb_address"`
}

// SecurityConfig holds CORS and rate limiting settings.
type SecurityConfig struct {
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
	RateLimitEnabled   bool     `mapstructure:"rate_limit_enabled"`
	RateLimitRPM       int      `mapstructure:"rate_limit_rpm"`
}

// SchedulerConfig holds resource scheduling configuration.
type SchedulerConfig struct {
	// CPUOvercommitRatio allows scheduling more vCPUs than physical cores.
	// A ratio of 4.0 means each physical core can host 4 vCPUs. Default: 1.0 (no overcommit).
	CPUOvercommitRatio float64 `mapstructure:"cpu_overcommit_ratio"`
	// RAMOvercommitRatio allows scheduling more RAM than physical memory.
	// A ratio of 1.5 means 64GB physical RAM can host 96GB of VM RAM. Default: 1.0 (no overcommit).
	RAMOvercommitRatio float64 `mapstructure:"ram_overcommit_ratio"`
	// DiskOvercommitRatio for thin-provisioned storage. Default: 1.0.
	DiskOvercommitRatio float64 `mapstructure:"disk_overcommit_ratio"`
}

// GatewayConfig holds service endpoint configuration for the API gateway.
// In monolithic mode all services share the same host:port, but in distributed
// mode each service can be addressed independently.
type GatewayConfig struct {
	IdentityHost  string `mapstructure:"identity_host"`
	IdentityPort  int    `mapstructure:"identity_port"`
	NetworkHost   string `mapstructure:"network_host"`
	NetworkPort   int    `mapstructure:"network_port"`
	SchedulerHost string `mapstructure:"scheduler_host"`
	SchedulerPort int    `mapstructure:"scheduler_port"`
	ComputeHost   string `mapstructure:"compute_host"`
	ComputePort   int    `mapstructure:"compute_port"`
}

// LoggingConfig holds log output settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// ModulesConfig controls which optional modules are enabled.
type ModulesConfig struct {
	EnableEvent      bool `mapstructure:"enable_event"`
	EnableQuota      bool `mapstructure:"enable_quota"`
	EnableConfig     bool `mapstructure:"enable_config"`
	EnableDomain     bool `mapstructure:"enable_domain"`
	EnableTools      bool `mapstructure:"enable_tools"`
	EnableUsage      bool `mapstructure:"enable_usage"`
	EnableVPN        bool `mapstructure:"enable_vpn"`
	EnableBackup     bool `mapstructure:"enable_backup"`
	EnableAutoScale  bool `mapstructure:"enable_autoscale"`
	EnableStorage    bool `mapstructure:"enable_storage"`
	EnableTask       bool `mapstructure:"enable_task"`
	EnableTag        bool `mapstructure:"enable_tag"`
	EnableNotify     bool `mapstructure:"enable_notification"`
	EnableImage      bool `mapstructure:"enable_image"`
	EnableAPIDocs    bool `mapstructure:"enable_apidocs"`
	EnableDNS        bool `mapstructure:"enable_dns"`
	EnableObjStorage bool `mapstructure:"enable_objectstorage"`
	EnableOrch       bool `mapstructure:"enable_orchestration"`
	EnableHA         bool `mapstructure:"enable_ha"`
	EnableKMS        bool `mapstructure:"enable_kms"`
	EnableRateLimit  bool `mapstructure:"enable_ratelimit"`
	EnableEncryption bool `mapstructure:"enable_encryption"`
	EnableCaaS       bool `mapstructure:"enable_caas"`
	EnableAudit      bool `mapstructure:"enable_audit"`
	EnableDR         bool `mapstructure:"enable_dr"`
	EnableBareMetal  bool `mapstructure:"enable_baremetal"`
	EnableCatalog    bool `mapstructure:"enable_catalog"`
	EnableSelfHeal   bool `mapstructure:"enable_selfheal"`
	EnableRegistry   bool `mapstructure:"enable_registry"`
	EnableConfigCtr  bool `mapstructure:"enable_configcenter"`
	EnableEventBus   bool `mapstructure:"enable_eventbus"`
}

// Load reads configuration from:
// 1. Defaults (hard-coded below)
// 2. YAML config file (searches ./configs/, /etc/vc-stack/, $HOME/.vcstack/)
// 3. Environment variables (prefix mapping, overrides file values)
//
// configName is the base name without extension: "vc-management" or "vc-compute".
func Load(configName string) (*AppConfig, error) {
	v := viper.New()

	// --- Defaults ---
	setDefaults(v)

	// --- Config file ---
	v.SetConfigName(configName)
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/etc/vc-stack")
	v.AddConfigPath("$HOME/.vcstack")
	v.AddConfigPath(".")

	// Read config file (not fatal if missing — env vars are sufficient).
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config file error: %w", err)
		}
		// Config file not found — rely on env vars and defaults.
	}

	// --- Environment variable bindings ---
	bindEnvVars(v)

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config unmarshal error: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "15m")
	v.SetDefault("server.write_timeout", "60m")
	v.SetDefault("server.idle_timeout", "60m")
	v.SetDefault("server.gin_mode", "debug")

	// Database
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "vcstack")
	v.SetDefault("database.username", "vcstack")
	v.SetDefault("database.password", "")      // Must be set via env var or config file
	v.SetDefault("database.sslmode", "prefer") // "require" recommended for production
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("database.auto_migrate", true)

	// Identity
	v.SetDefault("identity.access_token_expires_in", "24h")
	v.SetDefault("identity.refresh_token_expires_in", "168h")

	// Network
	v.SetDefault("sdn_provider", "ovn")

	// Logging
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Modules — all enabled by default
	v.SetDefault("modules.enable_event", true)
	v.SetDefault("modules.enable_quota", true)
	v.SetDefault("modules.enable_config", true)
	v.SetDefault("modules.enable_domain", true)
	v.SetDefault("modules.enable_tools", true)
	v.SetDefault("modules.enable_usage", true)
	v.SetDefault("modules.enable_vpn", true)
	v.SetDefault("modules.enable_backup", true)
	v.SetDefault("modules.enable_autoscale", true)
	v.SetDefault("modules.enable_storage", true)
	v.SetDefault("modules.enable_task", true)
	v.SetDefault("modules.enable_tag", true)
	v.SetDefault("modules.enable_notification", true)
	v.SetDefault("modules.enable_image", true)
	v.SetDefault("modules.enable_apidocs", true)
	v.SetDefault("modules.enable_dns", true)
	v.SetDefault("modules.enable_objectstorage", true)
	v.SetDefault("modules.enable_orchestration", true)
	v.SetDefault("modules.enable_ha", true)
	v.SetDefault("modules.enable_kms", true)
	v.SetDefault("modules.enable_ratelimit", true)
	v.SetDefault("modules.enable_encryption", true)
	v.SetDefault("modules.enable_caas", true)
	v.SetDefault("modules.enable_audit", true)
	v.SetDefault("modules.enable_dr", true)
	v.SetDefault("modules.enable_baremetal", true)
	v.SetDefault("modules.enable_catalog", true)
	v.SetDefault("modules.enable_selfheal", true)
	v.SetDefault("modules.enable_registry", true)
	v.SetDefault("modules.enable_configcenter", true)
	v.SetDefault("modules.enable_eventbus", true)

	// Scheduler
	v.SetDefault("scheduler.cpu_overcommit_ratio", 1.0)
	v.SetDefault("scheduler.ram_overcommit_ratio", 1.0)
	v.SetDefault("scheduler.disk_overcommit_ratio", 1.0)

	// Gateway service endpoints (monolithic defaults)
	v.SetDefault("gateway.identity_host", "localhost")
	v.SetDefault("gateway.identity_port", 8080)
	v.SetDefault("gateway.network_host", "localhost")
	v.SetDefault("gateway.network_port", 8080)
	v.SetDefault("gateway.scheduler_host", "localhost")
	v.SetDefault("gateway.scheduler_port", 8080)
	v.SetDefault("gateway.compute_host", "localhost")
	v.SetDefault("gateway.compute_port", 8080)

	// etcd (distributed lock / leader election)
	// Empty endpoints = etcd disabled (single-instance mode).
	v.SetDefault("etcd.endpoints", []string{})
	v.SetDefault("etcd.dial_timeout", "5s")
	v.SetDefault("etcd.tls", false)

	// Redis (session / cache / rate limiting)
	// Empty addr + empty sentinel_addrs = Redis disabled.
	v.SetDefault("redis.addr", "")
	v.SetDefault("redis.master_name", "")
	v.SetDefault("redis.sentinel_addrs", []string{})
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 3)

	// Kafka (message bus)
	// Empty brokers = Kafka disabled.
	v.SetDefault("kafka.brokers", []string{})
	v.SetDefault("kafka.consumer_group", "vc-management")
	v.SetDefault("kafka.compression", "snappy")
	v.SetDefault("kafka.max_retries", 3)
	v.SetDefault("kafka.dlq_topic", "vc.system.dlq")
	v.SetDefault("kafka.dlq_enabled", true)
	v.SetDefault("kafka.batch_size", 100)
	v.SetDefault("kafka.batch_timeout", "1s")
}

// bindEnvVars maps environment variable names to config keys.
// This preserves backward compatibility with existing env var names.
func bindEnvVars(v *viper.Viper) {
	// Enable automatic env binding with VC_ prefix for new-style vars.
	v.SetEnvPrefix("VC")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicit bindings for legacy env var names (no prefix).
	_ = v.BindEnv("database.host", "DB_HOST")
	_ = v.BindEnv("database.port", "DB_PORT")
	_ = v.BindEnv("database.name", "DB_NAME")
	_ = v.BindEnv("database.username", "DB_USER")
	_ = v.BindEnv("database.password", "DB_PASS")
	_ = v.BindEnv("database.sslmode", "DB_SSLMODE")

	_ = v.BindEnv("identity.jwt_secret", "JWT_SECRET")
	_ = v.BindEnv("server.gin_mode", "GIN_MODE")
	_ = v.BindEnv("server.port", "VC_MANAGEMENT_PORT")

	_ = v.BindEnv("sdn_provider", "VC_SDN_PROVIDER")
	_ = v.BindEnv("bridge_mappings", "VC_BRIDGE_MAPPINGS")
	_ = v.BindEnv("network.ovn_nb_address", "OVN_NB_ADDRESS")
	_ = v.BindEnv("external_url", "EXTERNAL_URL")

	_ = v.BindEnv("management_tls", "VC_MANAGEMENT_TLS")
	_ = v.BindEnv("web_console_dir", "WEB_CONSOLE_DIR")

	_ = v.BindEnv("ceph_rgw_endpoint", "CEPH_RGW_ENDPOINT")
	_ = v.BindEnv("ceph_rgw_access_key", "CEPH_RGW_ACCESS_KEY")
	_ = v.BindEnv("ceph_rgw_secret_key", "CEPH_RGW_SECRET_KEY")

	_ = v.BindEnv("sentry_dsn", "SENTRY_DSN")
	_ = v.BindEnv("sentry_environment", "SENTRY_ENVIRONMENT")

	_ = v.BindEnv("scheduler.cpu_overcommit_ratio", "SCHEDULER_CPU_OVERCOMMIT")
	_ = v.BindEnv("scheduler.ram_overcommit_ratio", "SCHEDULER_RAM_OVERCOMMIT")
	_ = v.BindEnv("scheduler.disk_overcommit_ratio", "SCHEDULER_DISK_OVERCOMMIT")

	// Gateway service endpoints
	_ = v.BindEnv("gateway.identity_host", "GATEWAY_IDENTITY_HOST")
	_ = v.BindEnv("gateway.identity_port", "GATEWAY_IDENTITY_PORT")
	_ = v.BindEnv("gateway.network_host", "GATEWAY_NETWORK_HOST")
	_ = v.BindEnv("gateway.network_port", "GATEWAY_NETWORK_PORT")
	_ = v.BindEnv("gateway.scheduler_host", "GATEWAY_SCHEDULER_HOST")
	_ = v.BindEnv("gateway.scheduler_port", "GATEWAY_SCHEDULER_PORT")
	_ = v.BindEnv("gateway.compute_host", "GATEWAY_COMPUTE_HOST")
	_ = v.BindEnv("gateway.compute_port", "GATEWAY_COMPUTE_PORT")

	// etcd
	_ = v.BindEnv("etcd.endpoints", "ETCD_ENDPOINTS")
	_ = v.BindEnv("etcd.dial_timeout", "ETCD_DIAL_TIMEOUT")
	_ = v.BindEnv("etcd.username", "ETCD_USERNAME")
	_ = v.BindEnv("etcd.password", "ETCD_PASSWORD")

	// Redis
	_ = v.BindEnv("redis.addr", "REDIS_ADDR")
	_ = v.BindEnv("redis.master_name", "REDIS_MASTER_NAME")
	_ = v.BindEnv("redis.sentinel_addrs", "REDIS_SENTINEL_ADDRS")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("redis.db", "REDIS_DB")

	// Kafka
	_ = v.BindEnv("kafka.brokers", "KAFKA_BROKERS")
	_ = v.BindEnv("kafka.consumer_group", "KAFKA_CONSUMER_GROUP")
	_ = v.BindEnv("kafka.compression", "KAFKA_COMPRESSION")
}
