package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// Service
	ServiceName    string `envconfig:"SERVICE_NAME" required:"true"`
	ServiceVersion string `envconfig:"SERVICE_VERSION" default:"1.0.0"`
	Environment    string `envconfig:"ENVIRONMENT" default:"development"`

	// Server Ports
	GRpcPort       string `envconfig:"GRPC_PORT" required:"true"`
	WSPort         string `envconfig:"WS_PORT" required:"true"`
	HealthPort     string `envconfig:"HEALTH_PORT"`
	PrometheusPort string `envconfig:"PROMETHEUS_PORT"`

	// Database
	DatabaseDSN string `envconfig:"DATABASE_DSN" required:"true"`

	// Redis
	RedisAddr string `envconfig:"REDIS_ADDR" required:"true"`
	
	TemplateBasePath string `envconfig:"TEMPLATE_BASE_PATH" required:"true"`
	
	// Kafka
	KafkaBrokers       []string `envconfig:"KAFKA_BROKERS" required:"true" separator:","`
	KafkaConsumerGroup string   `envconfig:"KAFKA_CONSUMER_GROUP" default:"notification-service"`
	KafkaWorkers       int      `envconfig:"KAFKA_WORKERS" default:"4"`
	KafkaRetries       int      `envconfig:"KAFKA_RETRIES" default:"3"`

	// SMTP
	SMTPHost     string `envconfig:"SMTP_HOST" required:"true"`
	SMTPPort     int    `envconfig:"SMTP_PORT" default:"587"`
	SMTPUsername string `envconfig:"SMTP_USERNAME" required:"true"`
	SMTPPassword string `envconfig:"SMTP_PASSWORD" required:"true"`

	EmailRateLimit  int `envconfig:"SMTP_RATE_LIMIT_RPM" default:"60"`
	EmailBurstLimit int `envconfig:"SMTP_RATE_LIMIT_BURST" default:"5"`

	// JWT
	JWTSecret string `envconfig:"JWT_ACCESS_TOKEN_SECRET" required:"true"`

	// Logging
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat string `envconfig:"LOG_FORMAT" default:"json"`
	LogFile   string `envconfig:"LOG_FILE" default:""`

	// Observability
	LokiURL    string `envconfig:"LOKI_URL" default:"http://loki:3100"`
	JaegerHost string `envconfig:"JAEGER_HOST" default:"jaeger"`
	JaegerPort int    `envconfig:"JAEGER_PORT" default:"6831"`
}

func LoadConfig() (*Config, error) {
	if os.Getenv("DOCKER_ENV") != "true" {
		loadDotEnv()
	}

	var config Config

	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("failed to load env config: %w", err)
	}

	// Custom validation
	if err := validate(&config); err != nil {
		return nil, err
	}


	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}


// loadDotEnv loads .env in local dev only (optional).
func loadDotEnv() {

	// Avoid dependency in prod
	_ = os.Setenv("ENV_LOADED", "true")

	_ = godotenv.Load()
}


// validate performs extra validation.
func validate(config *Config) error {

	// Environment sanity
	if !isValidEnv(config.Environment) {
		return errors.New("invalid ENVIRONMENT value")
	}

	// JWT security
	if len(config.JWTSecret) < 16 {
		return errors.New("JWT_ACCESS_TOKEN_SECRET must be at least 16 characters")
	}

	// Kafka check
	if len(config.KafkaBrokers) == 0 {
		return errors.New("KAFKA_BROKERS cannot be empty")
	}

	// Redis format
	if !strings.Contains(config.RedisAddr, ":") {
		return errors.New("invalid REDIS_ADDR format")
	}

	return nil
}

func validateConfig(config *Config) error {
	if config.ServiceName == "" {
		return fmt.Errorf("SERVICE_NAME is required")
	}

	if config.DatabaseDSN == "" {
		return fmt.Errorf("DATABASE_DSN is required")
	}

	if config.RedisAddr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}

	if len(config.KafkaBrokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}

	if config.JWTSecret == "" {
		if config.Environment == "production" {
			return fmt.Errorf("JWT_SECRET must be set in production")
		}
	}

	if config.SMTPHost == "" || config.SMTPUsername == "" || config.SMTPPassword == "" {
		return fmt.Errorf("SMTP configuration is incomplete host: %s username: %s password: %s config: %#v", config.SMTPHost, config.SMTPUsername, config.SMTPPassword, config)
	}

	return nil
}


func isValidEnv(env string) bool {

	switch env {
	case "development", "staging", "production", "test":
		return true
	default:
		return false
	}
}