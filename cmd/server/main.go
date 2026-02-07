package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	di "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/bootstrap"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/config"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/health"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	diCfg := &di.Config{
		ServiceName:      cfg.ServiceName,
		ServiceVersion:   cfg.ServiceVersion,
		Environment:      cfg.Environment,
		DatabaseDSN:      cfg.DatabaseDSN,
		RedisAddr:        cfg.RedisAddr,
		SMTPHost:         cfg.SMTPHost,
		SMTPPort:         strconv.Itoa(cfg.SMTPPort),
		SMTPUsername:     cfg.SMTPUsername,
		SMTPPassword:     cfg.SMTPPassword,
		KafkaBrokers:     cfg.KafkaBrokers,
		KafkaGroupID:     cfg.KafkaConsumerGroup,
		GRPCAddress:      ":" + cfg.GRpcPort,
		WSPort:           cfg.WSPort,
		JaegerHost:       cfg.JaegerHost,
		JaegerPort:       strconv.Itoa(cfg.JaegerPort),
		EmailRateLimit:   float64(cfg.EmailRateLimit),
		EmailBurstLimit:  cfg.EmailBurstLimit,
		KafkaWorkers:     cfg.KafkaWorkers,
		KafkaRetries:     cfg.KafkaRetries,
		TemplateBasePath: cfg.TemplateBasePath,
		JWTSecret:        cfg.JWTSecret,
	}

	container, err := di.NewContainer(diCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize container: %v\n", err)
		os.Exit(1)
	}

	logger := container.Logger
	logger.Info("Starting notification service",
		zap.String("version", cfg.ServiceVersion),
		zap.String("environment", cfg.Environment))

	metrics.InitMetrics()

	healthChecker := health.NewHealthChecker(
		container.DB,
		nil, 
		logger,
		cfg.ServiceName,
		cfg.ServiceVersion,
	)
	go func() {
		if err := startHealthServer(cfg.HealthPort, healthChecker); err != nil {
			logger.Error("Health server failed", zap.Error(err))
		}
	}()

	// context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := container.Start(ctx); err != nil {
		logger.Fatal("Failed to start services", zap.Error(err))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown signal received, initiating graceful shutdown...")

	//shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := container.Shutdown(shutdownCtx); err != nil {
		logger.Error("Shutdown error", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Service stopped gracefully")
}

func startHealthServer(port string, healthChecker *health.HealthChecker) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthChecker.LivenessHandler)
	mux.HandleFunc("/ready", healthChecker.ReadinessHandler)
	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	fmt.Printf("Health check and metrics server starting on port %s\n", port)
	return server.ListenAndServe()
}
