package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration. It is assembled from environment
// variables (and optionally a config file) and consumed by every layer that
// needs runtime parameters. Keeping all configuration in one struct makes the
// dependency surface explicit and testable.
//
// mapstructure tags match viper's keys exactly (viper uses mapstructure for
// Unmarshal; without tags, underscored keys like local_dir would not match the
// CamelCase field LocalDir).
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Storage  StorageConfig  `mapstructure:"storage"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Embedder EmbedderConfig `mapstructure:"embedder"`
	Queue    QueueConfig    `mapstructure:"queue"`
}

type ServerConfig struct {
	Port            string `mapstructure:"port"`
	ShutdownTimeout int    `mapstructure:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

func (d DatabaseConfig) DSN() string {
	return "host=" + d.Host +
		" port=" + d.Port +
		" user=" + d.User +
		" password=" + d.Password +
		" dbname=" + d.Name +
		" sslmode=" + d.SSLMode +
		" TimeZone=UTC"
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type StorageConfig struct {
	Provider   string `mapstructure:"provider"`
	LocalDir   string `mapstructure:"local_dir"`
	Endpoint   string `mapstructure:"endpoint"`
	AccessKey  string `mapstructure:"access_key"`
	SecretKey  string `mapstructure:"secret_key"`
	Bucket     string `mapstructure:"bucket"`
	UseSSL     bool   `mapstructure:"use_ssl"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
	Issuer      string `mapstructure:"issuer"`
}

type LLMConfig struct {
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
	Model    string `mapstructure:"model"`
}

type EmbedderConfig struct {
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
	Model    string `mapstructure:"model"`
	Dim      int    `mapstructure:"dim"`
}

type QueueConfig struct {
	Provider    string `mapstructure:"provider"`
	Concurrency int    `mapstructure:"concurrency"`
}

// Load reads configuration from environment variables. Defaults are chosen so
// the application runs out-of-the-box with the provided docker-compose stack.
//
// We set defaults for every key, then BindEnv each one explicitly. Viper's
// AutomaticEnv is unreliable with Unmarshal: it lazily substitutes env vars
// only on dotted-key lookups, so mapstructure-backed Unmarshal can miss them.
// Explicit BindEnv makes the env→field mapping deterministic.
func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Server
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.shutdown_timeout", 15)

	// Database
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "5432")
	v.SetDefault("database.user", "kn")
	v.SetDefault("database.password", "kn")
	v.SetDefault("database.name", "kn")
	v.SetDefault("database.sslmode", "disable")

	// Redis
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// Storage
	v.SetDefault("storage.provider", "local")
	v.SetDefault("storage.local_dir", "./uploads")
	v.SetDefault("storage.endpoint", "localhost:9000")
	v.SetDefault("storage.access_key", "minioadmin")
	v.SetDefault("storage.secret_key", "minioadmin")
	v.SetDefault("storage.bucket", "kn-documents")
	v.SetDefault("storage.use_ssl", false)

	// JWT
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expire_hours", 24)
	v.SetDefault("jwt.issuer", "kn-system")

	// LLM
	v.SetDefault("llm.provider", "mock")
	v.SetDefault("llm.api_key", "")
	v.SetDefault("llm.base_url", "https://api.openai.com/v1")
	v.SetDefault("llm.model", "gpt-4o-mini")

	// Embedder
	v.SetDefault("embedder.provider", "mock")
	v.SetDefault("embedder.api_key", "")
	v.SetDefault("embedder.base_url", "https://api.openai.com/v1")
	v.SetDefault("embedder.model", "text-embedding-3-small")
	v.SetDefault("embedder.dim", 1536)

	// Queue
	v.SetDefault("queue.provider", "memory")
	v.SetDefault("queue.concurrency", 4)

	// Explicit env bindings so Unmarshal picks them up. BindEnv with the env
	// name makes the mapping explicit (the env-key replacer only applies to
	// AutomaticEnv lookups, not to BindEnv, so we pass the names directly).
	type bind struct{ key, env string }
	binds := []bind{
		{"server.port", "SERVER_PORT"},
		{"server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT"},
		{"database.host", "DATABASE_HOST"},
		{"database.port", "DATABASE_PORT"},
		{"database.user", "DATABASE_USER"},
		{"database.password", "DATABASE_PASSWORD"},
		{"database.name", "DATABASE_NAME"},
		{"database.sslmode", "DATABASE_SSLMODE"},
		{"redis.addr", "REDIS_ADDR"},
		{"redis.password", "REDIS_PASSWORD"},
		{"redis.db", "REDIS_DB"},
		{"storage.provider", "STORAGE_PROVIDER"},
		{"storage.local_dir", "STORAGE_LOCAL_DIR"},
		{"storage.endpoint", "STORAGE_ENDPOINT"},
		{"storage.access_key", "STORAGE_ACCESS_KEY"},
		{"storage.secret_key", "STORAGE_SECRET_KEY"},
		{"storage.bucket", "STORAGE_BUCKET"},
		{"storage.use_ssl", "STORAGE_USE_SSL"},
		{"jwt.secret", "JWT_SECRET"},
		{"jwt.expire_hours", "JWT_EXPIRE_HOURS"},
		{"jwt.issuer", "JWT_ISSUER"},
		{"llm.provider", "LLM_PROVIDER"},
		{"llm.api_key", "LLM_API_KEY"},
		{"llm.base_url", "LLM_BASE_URL"},
		{"llm.model", "LLM_MODEL"},
		{"embedder.provider", "EMBEDDER_PROVIDER"},
		{"embedder.api_key", "EMBEDDER_API_KEY"},
		{"embedder.base_url", "EMBEDDER_BASE_URL"},
		{"embedder.model", "EMBEDDER_MODEL"},
		{"embedder.dim", "EMBEDDER_DIM"},
		{"queue.provider", "QUEUE_PROVIDER"},
		{"queue.concurrency", "QUEUE_CONCURRENCY"},
	}
	for _, b := range binds {
		_ = v.BindEnv(b.key, b.env)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
