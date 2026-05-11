package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Mode        string
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Monitor     MonitorConfig
	Scheduler   SchedulerConfig
	Kubernetes  KubernetesConfig
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	MaxBodySize  int64
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConn  int
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
	RefreshExp time.Duration
}

type MonitorConfig struct {
	Enabled         bool
	PrometheusURL   string
	GrafanaURL      string
	ScrapeInterval  time.Duration
	RetentionPeriod time.Duration
}

type SchedulerConfig struct {
	Enabled         bool
	Type            string
	MaxRetry        int
	QueueSize       int
	Workers         int
	PendingTimeout  time.Duration
}

type KubernetesConfig struct {
	APIServer string
	KubeConfig string
	InCluster bool
	QPS       float32
	Burst     int
	Timeout   time.Duration
}

type AgentConfig struct {
	Port    int
	Server  string
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/edgehub/")
	viper.AddConfigPath("$HOME/.edgehub")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	viper.SetDefault("mode", "debug")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.max_body_size", 10485760)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_conn", 100)
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.pool_size", 100)
	viper.SetDefault("jwt.expiration", 24)
	viper.SetDefault("jwt.refresh_exp", 720)
	viper.SetDefault("monitor.enabled", true)
	viper.SetDefault("monitor.scrape_interval", 15)
	viper.SetDefault("monitor.retention_period", 15)
	viper.SetDefault("scheduler.enabled", true)
	viper.SetDefault("scheduler.type", "kueue")
	viper.SetDefault("scheduler.max_retry", 3)
	viper.SetDefault("scheduler.queue_size", 1000)
	viper.SetDefault("scheduler.workers", 10)
	viper.SetDefault("kubernetes.qps", 50)
	viper.SetDefault("kubernetes.burst", 100)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(fmt.Errorf("failed to read config file: %w", err))
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %w", err))
	}

	cfg.loadEnvOverrides()

	return &cfg
}

func (c *Config) loadEnvOverrides() {
	if host := os.Getenv("DB_HOST"); host != "" {
		c.Database.Host = host
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		c.Database.Password = password
	}
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		c.Redis.Host = redisHost
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		c.Redis.Password = redisPassword
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		c.JWT.Secret = jwtSecret
	}
	if prometheusURL := os.Getenv("PROMETHEUS_URL"); prometheusURL != "" {
		c.Monitor.PrometheusURL = prometheusURL
	}
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		c.Kubernetes.KubeConfig = kubeconfig
	}
}
