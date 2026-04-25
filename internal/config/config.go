package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Asynq        AsynqConfig        `mapstructure:"asynq"`
	Admin        AdminConfig        `mapstructure:"admin"`
	OpenAIImage2 OpenAIImage2Config `mapstructure:"openai_image2"`
	ResultTTL    time.Duration      `mapstructure:"result_ttl"`
	RefImgTTL    time.Duration      `mapstructure:"refimg_ttl"`
	CreditTTL    time.Duration      `mapstructure:"credit_ttl"`
	TaskTTL      time.Duration      `mapstructure:"task_ttl"`
	SessionTTL   time.Duration      `mapstructure:"session_ttl"`
	Log          LogConfig          `mapstructure:"log"`
}

type ServerConfig struct {
	Addr         string        `mapstructure:"addr"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

type AsynqConfig struct {
	RedisAddr     string         `mapstructure:"redis_addr"`
	RedisPassword string         `mapstructure:"redis_password"`
	Concurrency   int            `mapstructure:"concurrency"`
	Queues        map[string]int `mapstructure:"queues"`
}

type AdminConfig struct {
	Token string `mapstructure:"token"`
}

type OpenAIImage2Config struct {
	APIKey       string `mapstructure:"api_key"`
	Endpoint     string `mapstructure:"endpoint"`
	Model        string `mapstructure:"model"`
	DefaultSize  string `mapstructure:"default_size"`
	MaxImages    int    `mapstructure:"max_images"`
	SupportsRefs bool   `mapstructure:"supports_references"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load(path string) (*Config, error) {
	// Load .env file if it exists (same directory as config.yaml)
	if dir := filepath.Dir(path); dir != "" {
		envFile := filepath.Join(dir, ".env")
		if _, err := os.Stat(envFile); err == nil {
			envBytes, _ := os.ReadFile(envFile)
			for _, line := range strings.Split(string(envBytes), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if i := strings.Index(line, "="); i > 0 {
					os.Setenv(strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]))
				}
			}
		}
	}

	v := viper.New()

	v.SetConfigFile(path)
	v.AutomaticEnv()
	v.BindEnv("admin.token", "ADMIN_TOKEN")
	v.BindEnv("openai_image2.api_key", "OPENAI_API_KEY")
	v.BindEnv("openai_image2.model", "OPENAI_IMAGE2_MODEL")

	v.SetDefault("server.addr", ":8080")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "60s")
	v.SetDefault("server.idle_timeout", "120s")
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 20)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("asynq.concurrency", 4)
	v.SetDefault("asynq.queues", map[string]int{"critical": 6, "default": 4})
	v.SetDefault("openai_image2.endpoint", "https://api.openai.com")
	v.SetDefault("openai_image2.model", "openai-image2")
	v.SetDefault("openai_image2.default_size", "1024x1024")
	v.SetDefault("openai_image2.max_images", 4)
	v.SetDefault("openai_image2.supports_references", false)
	v.SetDefault("result_ttl", "2h")
	v.SetDefault("refimg_ttl", "30m")
	v.SetDefault("credit_ttl", "168h")
	v.SetDefault("task_ttl", "24h")
	v.SetDefault("session_ttl", "720h")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if cfg.Asynq.RedisAddr == "" {
		cfg.Asynq.RedisAddr = cfg.Redis.Addr
	}
	if cfg.Asynq.RedisPassword == "" {
		cfg.Asynq.RedisPassword = cfg.Redis.Password
	}

	return &cfg, nil
}

func DefaultAsynqTimeout() time.Duration {
	return 120 * time.Second
}

func DefaultTaskRetention() time.Duration {
	return 24 * time.Hour
}
