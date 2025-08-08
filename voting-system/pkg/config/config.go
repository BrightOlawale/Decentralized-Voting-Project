package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Blockchain BlockchainConfig `mapstructure:"blockchain"`
	Biometric  BiometricConfig  `mapstructure:"biometric"`
	Hardware   HardwareConfig   `mapstructure:"hardware"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Security   SecurityConfig   `mapstructure:"security"`
	API        APIConfig        `mapstructure:"api"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         string        `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"` // gin mode: debug, release, test
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	TLS          TLSConfig     `mapstructure:"tls"`
}

// TLSConfig holds TLS/SSL configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Type         string        `mapstructure:"type"` // postgres, sqlite, mysql
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	User         string        `mapstructure:"user"`
	Password     string        `mapstructure:"password"`
	DBName       string        `mapstructure:"dbname"`
	Path         string        `mapstructure:"path"`    // For SQLite
	SSLMode      string        `mapstructure:"sslmode"` // For PostgreSQL
	MaxOpenConns int           `mapstructure:"max_open_conns"`
	MaxIdleConns int           `mapstructure:"max_idle_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
}

// BlockchainConfig holds blockchain-related configuration
type BlockchainConfig struct {
	NetworkURL      string        `mapstructure:"network_url"`
	ContractAddress string        `mapstructure:"contract_address"`
	PrivateKey      string        `mapstructure:"private_key"`
	ChainID         int64         `mapstructure:"chain_id"`
	GasLimit        uint64        `mapstructure:"gas_limit"`
	GasPrice        int64         `mapstructure:"gas_price"`
	SyncInterval    time.Duration `mapstructure:"sync_interval"`
	RetryInterval   time.Duration `mapstructure:"retry_interval"`
	MaxRetries      int           `mapstructure:"max_retries"`
	ConfirmBlocks   int           `mapstructure:"confirm_blocks"`
}

// BiometricConfig holds biometric verification configuration
type BiometricConfig struct {
	FingerprintDevice string  `mapstructure:"fingerprint_device"`
	QualityThreshold  float64 `mapstructure:"quality_threshold"`
	MatchThreshold    float64 `mapstructure:"match_threshold"`
	MaxAttempts       int     `mapstructure:"max_attempts"`
	TemplateFormat    string  `mapstructure:"template_format"`
	Enabled           bool    `mapstructure:"enabled"`
}

// HardwareConfig holds hardware interface configuration
type HardwareConfig struct {
	DisplayDevice string `mapstructure:"display_device"`
	PrinterDevice string `mapstructure:"printer_device"`
	CardReader    string `mapstructure:"card_reader"`
	CameraDevice  string `mapstructure:"camera_device"`
	StatusLEDs    bool   `mapstructure:"status_leds"`
	Buzzer        bool   `mapstructure:"buzzer"`
}

// EncryptionConfig holds encryption-related configuration
type EncryptionConfig struct {
	Key         string `mapstructure:"key"`
	Algorithm   string `mapstructure:"algorithm"` // AES-256-GCM, ChaCha20-Poly1305
	KeyRotation bool   `mapstructure:"key_rotation"`
}

// RedisConfig holds Redis configuration for caching and sessions
type RedisConfig struct {
	Addr         string        `mapstructure:"addr"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	Enabled      bool          `mapstructure:"enabled"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`  // debug, info, warn, error
	Format     string `mapstructure:"format"` // json, text
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"` // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // days
	Compress   bool   `mapstructure:"compress"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret"`
	JWTExpiration       time.Duration `mapstructure:"jwt_expiration"`
	SessionTimeout      time.Duration `mapstructure:"session_timeout"`
	MaxLoginAttempts    int           `mapstructure:"max_login_attempts"`
	LockoutDuration     time.Duration `mapstructure:"lockout_duration"`
	PasswordMinLength   int           `mapstructure:"password_min_length"`
	RequireStrongPasswd bool          `mapstructure:"require_strong_password"`
	EnableTwoFA         bool          `mapstructure:"enable_2fa"`
}

// APIConfig holds API-related configuration
type APIConfig struct {
	RateLimit     int           `mapstructure:"rate_limit"` // requests per minute
	BurstLimit    int           `mapstructure:"burst_limit"`
	Timeout       time.Duration `mapstructure:"timeout"`
	CORS          CORSConfig    `mapstructure:"cors"`
	Versioning    bool          `mapstructure:"versioning"`
	Documentation bool          `mapstructure:"documentation"` // Enable API docs
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	// Set default values
	setDefaults()

	// Set config file path
	viper.SetConfigFile(configPath)

	// Allow environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("VOTING")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; use defaults and env vars
			fmt.Printf("Warning: Config file not found at %s, using defaults\n", configPath)
		} else {
			return nil, fmt.Errorf("error reading config file: %v", err)
		}
	}

	// Override with environment variables
	overrideWithEnvVars()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %v", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "60s")

	// Database defaults
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.path", "./voting.db")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_lifetime", "5m")

	// Blockchain defaults
	viper.SetDefault("blockchain.network_url", "http://localhost:8545")
	viper.SetDefault("blockchain.chain_id", 1337)
	viper.SetDefault("blockchain.gas_limit", 3000000)
	viper.SetDefault("blockchain.gas_price", 20000000000) // 20 Gwei
	viper.SetDefault("blockchain.sync_interval", "30s")
	viper.SetDefault("blockchain.retry_interval", "30s")
	viper.SetDefault("blockchain.max_retries", 3)
	viper.SetDefault("blockchain.confirm_blocks", 1)

	// Biometric defaults
	viper.SetDefault("biometric.quality_threshold", 0.8)
	viper.SetDefault("biometric.match_threshold", 0.85)
	viper.SetDefault("biometric.max_attempts", 3)
	viper.SetDefault("biometric.template_format", "iso")
	viper.SetDefault("biometric.enabled", true)

	// Hardware defaults
	viper.SetDefault("hardware.status_leds", true)
	viper.SetDefault("hardware.buzzer", true)

	// Encryption defaults
	viper.SetDefault("encryption.algorithm", "AES-256-GCM")
	viper.SetDefault("encryption.key_rotation", false)

	// Redis defaults
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.dial_timeout", "5s")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")
	viper.SetDefault("redis.enabled", false)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("logging.file", "./logs/app.log")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
	viper.SetDefault("logging.compress", true)

	// Security defaults
	viper.SetDefault("security.jwt_expiration", "24h")
	viper.SetDefault("security.session_timeout", "2h")
	viper.SetDefault("security.max_login_attempts", 5)
	viper.SetDefault("security.lockout_duration", "15m")
	viper.SetDefault("security.password_min_length", 8)
	viper.SetDefault("security.require_strong_password", true)
	viper.SetDefault("security.enable_2fa", false)

	// API defaults
	viper.SetDefault("api.rate_limit", 100)
	viper.SetDefault("api.burst_limit", 200)
	viper.SetDefault("api.timeout", "30s")
	viper.SetDefault("api.versioning", true)
	viper.SetDefault("api.documentation", true)

	// CORS defaults
	viper.SetDefault("api.cors.allowed_origins", []string{"*"})
	viper.SetDefault("api.cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("api.cors.allowed_headers", []string{"*"})
	viper.SetDefault("api.cors.allow_credentials", true)
	viper.SetDefault("api.cors.max_age", 86400)
}

// overrideWithEnvVars overrides config with specific environment variables
func overrideWithEnvVars() {
	// Critical environment variables that should always override config
	envMappings := map[string]string{
		"PRIVATE_KEY":      "blockchain.private_key",
		"CONTRACT_ADDRESS": "blockchain.contract_address",
		"NETWORK_URL":      "blockchain.network_url",
		"DATABASE_URL":     "database.url",
		"DB_PASSWORD":      "database.password",
		"DB_USER":          "database.user",
		"ENCRYPTION_KEY":   "encryption.key",
		"JWT_SECRET":       "security.jwt_secret",
		"ADMIN_USERNAME":   "admin.username",
		"ADMIN_PASSWORD":   "admin.password",
		"REDIS_URL":        "redis.addr",
		"REDIS_PASSWORD":   "redis.password",
	}

	for envVar, configKey := range envMappings {
		if value := os.Getenv(envVar); value != "" {
			viper.Set(configKey, value)
		}
	}
}

// validateConfig validates the loaded configuration
func validateConfig(config *Config) error {
	// Validate required fields
	if config.Blockchain.ContractAddress == "" {
		return fmt.Errorf("blockchain contract address is required")
	}

	if config.Blockchain.PrivateKey == "" {
		return fmt.Errorf("blockchain private key is required")
	}

	if config.Encryption.Key == "" {
		return fmt.Errorf("encryption key is required")
	}

	if len(config.Encryption.Key) < 32 {
		return fmt.Errorf("encryption key must be at least 32 characters")
	}

	if config.Security.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	// Validate port range
	if config.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	// Validate database configuration
	if config.Database.Type == "postgres" {
		if config.Database.Host == "" || config.Database.User == "" {
			return fmt.Errorf("postgres requires host and user")
		}
	} else if config.Database.Type == "sqlite" {
		if config.Database.Path == "" {
			return fmt.Errorf("sqlite requires path")
		}
	}

	// Validate biometric thresholds
	if config.Biometric.QualityThreshold < 0 || config.Biometric.QualityThreshold > 1 {
		return fmt.Errorf("biometric quality threshold must be between 0 and 1")
	}

	if config.Biometric.MatchThreshold < 0 || config.Biometric.MatchThreshold > 1 {
		return fmt.Errorf("biometric match threshold must be between 0 and 1")
	}

	// Validate blockchain configuration
	if config.Blockchain.GasLimit == 0 {
		config.Blockchain.GasLimit = 3000000 // Set default
	}

	if config.Blockchain.ChainID == 0 {
		config.Blockchain.ChainID = 1337 // Set default for development
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	switch c.Database.Type {
	case "postgres":
		sslMode := c.Database.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Database.Host, c.Database.Port, c.Database.User,
			c.Database.Password, c.Database.DBName, sslMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.Database.User, c.Database.Password, c.Database.Host,
			c.Database.Port, c.Database.DBName)
	case "sqlite":
		return c.Database.Path
	default:
		return ""
	}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Mode == "debug" || c.Server.Mode == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Mode == "release" || c.Server.Mode == "production"
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

// SanitizeForLogging returns a copy of the config with sensitive data redacted
func (c *Config) SanitizeForLogging() *Config {
	sanitized := *c

	// Redact sensitive information
	if sanitized.Database.Password != "" {
		sanitized.Database.Password = "[REDACTED]"
	}

	if sanitized.Blockchain.PrivateKey != "" {
		sanitized.Blockchain.PrivateKey = "[REDACTED]"
	}

	if sanitized.Encryption.Key != "" {
		sanitized.Encryption.Key = "[REDACTED]"
	}

	if sanitized.Security.JWTSecret != "" {
		sanitized.Security.JWTSecret = "[REDACTED]"
	}

	if sanitized.Redis.Password != "" {
		sanitized.Redis.Password = "[REDACTED]"
	}

	return &sanitized
}

// LoadConfigFromEnv loads configuration primarily from environment variables
func LoadConfigFromEnv() (*Config, error) {
	config := &Config{}

	// Server configuration
	config.Server.Host = getEnvOrDefault("SERVER_HOST", "0.0.0.0")
	config.Server.Port = getEnvOrDefault("SERVER_PORT", "8080")
	config.Server.Mode = getEnvOrDefault("GIN_MODE", "debug")

	// Database configuration
	config.Database.Type = getEnvOrDefault("DB_TYPE", "sqlite")
	config.Database.Host = getEnvOrDefault("DB_HOST", "localhost")
	config.Database.Port = getEnvInt("DB_PORT", 5432)
	config.Database.User = getEnvOrDefault("DB_USER", "voting_user")
	config.Database.Password = os.Getenv("DB_PASSWORD")
	config.Database.DBName = getEnvOrDefault("DB_NAME", "voting_system")
	config.Database.Path = getEnvOrDefault("DB_PATH", "./voting.db")

	// Blockchain configuration
	config.Blockchain.NetworkURL = getEnvOrDefault("NETWORK_URL", "http://localhost:8545")
	config.Blockchain.ContractAddress = os.Getenv("CONTRACT_ADDRESS")
	config.Blockchain.PrivateKey = os.Getenv("PRIVATE_KEY")
	config.Blockchain.ChainID = getEnvInt64("CHAIN_ID", 1337)
	config.Blockchain.GasLimit = getEnvUint64("GAS_LIMIT", 3000000)
	config.Blockchain.GasPrice = getEnvInt64("GAS_PRICE", 20000000000)

	// Encryption configuration
	config.Encryption.Key = os.Getenv("ENCRYPTION_KEY")
	config.Encryption.Algorithm = getEnvOrDefault("ENCRYPTION_ALGORITHM", "AES-256-GCM")

	// Security configuration
	config.Security.JWTSecret = os.Getenv("JWT_SECRET")

	// Logging configuration
	config.Logging.Level = getEnvOrDefault("LOG_LEVEL", "info")
	config.Logging.Format = getEnvOrDefault("LOG_FORMAT", "text")
	config.Logging.File = getEnvOrDefault("LOG_FILE", "./logs/app.log")

	// Validate the configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// Helper functions for environment variable parsing
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvUint64(key string, defaultValue uint64) uint64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseUint(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
