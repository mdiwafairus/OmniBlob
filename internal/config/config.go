package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server" mapstructure:"server"`
	Auth      AuthConfig      `yaml:"auth" mapstructure:"auth"`
	Storage   StorageConfig   `yaml:"storage" mapstructure:"storage"`
	Migration MigrationConfig `yaml:"migration" mapstructure:"migration"`
	Reconciliation ReconciliationConfig `yaml:"reconciliation" mapstructure:"reconciliation"`
	Database  DatabaseConfig  `yaml:"database" mapstructure:"database"`
	Security  SecurityConfig  `yaml:"security" mapstructure:"security"`
}

type AuthConfig struct {
	AccessKey string `yaml:"access_key" mapstructure:"access_key"`
	SecretKey string `yaml:"secret_key" mapstructure:"secret_key"`
}


type ClientConfig struct {
	Name           string   `yaml:"name" mapstructure:"name"`
	ApiKey         string   `yaml:"api_key" mapstructure:"api_key"`
	User           string   `yaml:"user" mapstructure:"user"`
	Password       string   `yaml:"password" mapstructure:"password"`
	WhitelistIPs   []string `yaml:"whitelist_ips" mapstructure:"whitelist_ips"`
	AllowedModules []string `yaml:"allowed_modules" mapstructure:"allowed_modules"`
}

type ServerConfig struct {
	Host            string         `yaml:"host" mapstructure:"host"`
	Port            int            `yaml:"port" mapstructure:"port"`
	ReadTimeoutSec  int            `yaml:"read_timeout_sec" mapstructure:"read_timeout_sec"`
	WriteTimeoutSec int            `yaml:"write_timeout_sec" mapstructure:"write_timeout_sec"`
	MaxUploadSizeMB int            `yaml:"max_upload_size_mb" mapstructure:"max_upload_size_mb"`
	ApiKey          string         `yaml:"api_key" mapstructure:"api_key"`
	ApiUser         string         `yaml:"api_user" mapstructure:"api_user"`
	ApiPass         string         `yaml:"api_pass" mapstructure:"api_pass"`
	AllowedExtensions []string       `yaml:"allowed_extensions" mapstructure:"allowed_extensions"`
	RateLimitRPM      int            `yaml:"rate_limit_rpm" mapstructure:"rate_limit_rpm"`
	Clients           []ClientConfig `yaml:"clients" mapstructure:"clients"`
}

type SecurityConfig struct {
	Enabled          bool     `yaml:"enabled" mapstructure:"enabled"`
	Mode             string   `yaml:"mode" mapstructure:"mode"`
	AllowedSocketIPs []string `yaml:"allowed_socket_ips" mapstructure:"allowed_socket_ips"`
	AuthType         string   `yaml:"auth_type" mapstructure:"auth_type"`
}

type StorageConfig struct {
	RootPath     string `yaml:"root_path" mapstructure:"root_path"`
	LegacyPath   string `yaml:"legacy_path" mapstructure:"legacy_path"`
	ShardingType string `yaml:"sharding_type" mapstructure:"sharding_type"`
}

type MigrationConfig struct {
	Enabled     bool `yaml:"enabled" mapstructure:"enabled"`
	BatchSize   int  `yaml:"batch_size" mapstructure:"batch_size"`
	IntervalSec int  `yaml:"interval_sec" mapstructure:"interval_sec"`
	WorkerCount int  `yaml:"worker_count" mapstructure:"worker_count"`
}

type ReconciliationConfig struct {
	Enabled     bool `yaml:"enabled" mapstructure:"enabled"`
	BatchSize   int  `yaml:"batch_size" mapstructure:"batch_size"`
	IntervalSec int  `yaml:"interval_sec" mapstructure:"interval_sec"`
}

type DatabaseConfig struct {
	Host        string `yaml:"host" mapstructure:"host"`
	Port        int    `yaml:"port" mapstructure:"port"`
	Username    string `yaml:"username" mapstructure:"username"`
	Password    string `yaml:"password" mapstructure:"password"`
	Name        string `yaml:"name" mapstructure:"name"`
	SSLMode     string `yaml:"sslmode" mapstructure:"sslmode"`
	MaxOpenConn int    `yaml:"max_open_conn" mapstructure:"max_open_conn"`
	MinOpenConn int    `yaml:"min_open_conn" mapstructure:"min_open_conn"`
	MaxIdleTime int    `yaml:"max_idle_time" mapstructure:"max_idle_time"`
	MaxLifeTime int    `yaml:"max_life_time" mapstructure:"max_life_time"`
}

// loadDotEnv loads key-value pairs from a .env file into the environment if not already set.
func loadDotEnv(envPath string) {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return // File .env is optional
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}
}

// LoadConfig reads the YAML configuration file from the given path and applies any environment / .env overrides.
func LoadConfig(path string) (*Config, error) {
	// 1. Auto-load .env if present in working directory
	loadDotEnv(".env")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config yaml: %w", err)
	}

	// 2. Environment variable overrides (from .env or OS / Systemd)
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		if p, err := strconv.Atoi(dbPort); err == nil {
			cfg.Database.Port = p
		}
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		cfg.Database.Username = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		cfg.Database.Password = dbPass
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		cfg.Database.Name = dbName
	}
	if dbSSL := os.Getenv("DB_SSLMODE"); dbSSL != "" {
		cfg.Database.SSLMode = dbSSL
	}
	if srvPort := os.Getenv("PORT"); srvPort != "" {
		if p, err := strconv.Atoi(srvPort); err == nil {
			cfg.Server.Port = p
		}
	}
	if srvHost := os.Getenv("HOST"); srvHost != "" {
		cfg.Server.Host = srvHost
	}
	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		cfg.Server.ApiKey = apiKey
	}
	if apiUser := os.Getenv("API_USER"); apiUser != "" {
		cfg.Server.ApiUser = apiUser
	}
	if apiPass := os.Getenv("API_PASS"); apiPass != "" {
		cfg.Server.ApiPass = apiPass
	}
	if migrationEnabled := os.Getenv("MIGRATION_ENABLED"); migrationEnabled != "" {
		if b, err := strconv.ParseBool(migrationEnabled); err == nil {
			cfg.Migration.Enabled = b
		}
	}
	if reconciliationEnabled := os.Getenv("RECONCILIATION_ENABLED"); reconciliationEnabled != "" {
		if b, err := strconv.ParseBool(reconciliationEnabled); err == nil {
			cfg.Reconciliation.Enabled = b
		}
	}

	return &cfg, nil
}

