package di

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/events"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/services"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_events "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/events"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	database "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/database/gorm"
	email "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/email"
	grpc_server "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/grpc"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/kafka"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/notification"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/observability/logging"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/observability/tracing"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/ratelimit"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/redis"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/template"
	grpc_interface "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/interfaces/grpc"
	websocket_interface "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/interfaces/websocket"
	ws "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/interfaces/websocket"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Container struct {
	// Config
	Config *Config

	// Infrastructure
	Logger *zap.Logger
	Tracer *tracing.Tracer
	DB     *database.DB
	Cache  ports.Cache

	// Repositories
	NotificationRepo repository.NotificationRepository
	OTPRepo          repository.OTPRepository

	// Infrastructure services
	NotificationSender ports.NotificationSender
	RateLimiter        ports.RateLimiter
	TemplateRenderer   ports.TemplateRenderer
	EmailSender        *email.EmailSender
	WSHub              *websocket_interface.Hub
	KafkaProducer      *kafka.Producer
	KafkaConsumer      *kafka.Consumer

	// // EventHandlers
	// OTPRequestEventHandler     *events.OTPRequestEventHandler
	// ForgotPasswordEventHandler *events.ForgotPasswordEventHandler

	// Adaptors
	MessageBroker ports.MessageBroker

	// Application services
	NotificationService   *services.NotificationService
	EmailOTPService       *services.EmailOTPService
	ForgotPasswordService *services.ForgotPasswordService

	// Presentation
	GRPCServer *grpc_server.Server
}

type Config struct {
	// Service
	ServiceName    string
	ServiceVersion string
	Environment    string

	// Database
	DatabaseDSN string

	// Redis
	RedisAddr string

	// Template
	TemplateBasePath string

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string

	// Kafka
	KafkaBrokers []string
	KafkaGroupID string

	// gRPC
	GRPCAddress string

	// WebSocket
	WSPort string

	// Observability
	JaegerHost string
	JaegerPort string

	// Rate limiting
	EmailRateLimit  float64
	EmailBurstLimit int

	// Workers
	KafkaWorkers int
	KafkaRetries int

	// Security
	JWTSecret string
}

func NewContainer(cfg *Config) (*Container, error) {
	c := &Container{Config: cfg}

	if err := c.initObservability(); err != nil {
		return nil, fmt.Errorf("observability init failed: %w", err)
	}

	if err := c.initInfrastructure(); err != nil {
		return nil, fmt.Errorf("infrastructure init failed: %w", err)
	}

	if err := c.initRepositories(); err != nil {
		return nil, fmt.Errorf("repositories init failed: %w", err)
	}

	if err := c.initServices(); err != nil {
		return nil, fmt.Errorf("services init failed: %w", err)
	}
	if err := c.initEventHandlers(); err != nil {
		return nil, fmt.Errorf("event handlers init failed: %w", err)
	}

	if err := c.initPresentation(); err != nil {
		return nil, fmt.Errorf("presentation init failed: %w", err)
	}

	c.Logger.Info("Dependency injection container initialized successfully")
	return c, nil
}

func (c *Container) initObservability() error {
	// Logger
	logConfig := logging.LoggingConfig{
		LogLevel:       "info",
		LogFormat:      "json",
		ServiceName:    c.Config.ServiceName,
		ServiceVersion: c.Config.ServiceVersion,
		Environment:    c.Config.Environment,
	}
	logger, err := logging.NewLogger(logConfig)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	c.Logger = logger

	// Tracer
	tracingConfig := tracing.TracingConfig{
		JaegerHost:     c.Config.JaegerHost,
		JaegerPort:     c.Config.JaegerPort,
		ServiceName:    c.Config.ServiceName,
		ServiceVersion: c.Config.ServiceVersion,
		Environment:    c.Config.Environment,
	}
	tracer, err := tracing.NewTracer(tracingConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to create tracer: %w", err)
	}
	c.Tracer = tracer

	return nil
}

func (c *Container) initInfrastructure() error {
	db, err := database.NewDB(c.Config.DatabaseDSN, c.Logger)
	if err != nil {
		return fmt.Errorf("database init failed: %w", err)
	}
	c.DB = db

	if err := c.DB.AutoMigrate(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	cache, err := redis.NewRedisCache(
		c.Config.RedisAddr,
		c.Logger,
		c.Config.ServiceName,
	)
	if err != nil {
		return fmt.Errorf("cache init failed: %w", err)
	}
	c.Cache = cache

	renderer, err := template.NewTemplateRenderer(
		c.Config.TemplateBasePath,
		cache,
	)
	if err != nil {
		return fmt.Errorf("template renderer init failed: %w", err)
	}
	c.TemplateRenderer = renderer

	c.RateLimiter = ratelimit.NewRateLimiter(
		cache,
		int(c.Config.EmailRateLimit),
		time.Duration(c.Config.EmailBurstLimit)*time.Minute,
		c.Config.ServiceName,
	)

	emailConfig := email.DefaultEmailConfig(
		c.Config.SMTPHost,
		c.Config.SMTPPort,
		c.Config.SMTPUsername,
		c.Config.SMTPPassword,
	)
	emailSender, err := email.NewEmailSender(emailConfig, c.RateLimiter, c.Logger)
	if err != nil {
		return fmt.Errorf("email sender init failed: %w", err)
	}
	c.EmailSender = emailSender

	c.WSHub = websocket_interface.NewHub(c.Logger)

	producer, err := kafka.NewProducer(c.Config.KafkaBrokers, c.Logger)
	if err != nil {
		return fmt.Errorf("kafka producer init failed: %w", err)
	}
	c.KafkaProducer = producer

	// Kafka Consumer
	consumerConfig := kafka.DefaultConsumerConfig()
	consumerConfig.Workers = c.Config.KafkaWorkers
	consumerConfig.Retries = c.Config.KafkaRetries

	consumer, err := kafka.NewConsumer(c.Config.KafkaBrokers, c.Config.KafkaGroupID, c.Logger, consumerConfig)
	if err != nil {
		return fmt.Errorf("kafka consumer init failed: %w", err)
	}
	c.KafkaConsumer = consumer

	c.MessageBroker = kafka.NewMessageBrokerAdapter(c.KafkaProducer, consumer)

	return nil
}

func (c *Container) initRepositories() error {
	c.NotificationRepo = database.NewNotificationRepository(
		c.DB.Gorm(),
		c.Cache,
		c.Logger,
	)

	c.OTPRepo = redis.NewOTPRepository(
		c.Cache,
		c.Logger,
	)

	return nil
}

func (c *Container) initServices() error {
	c.EmailOTPService = services.NewEmailOTPService(
		c.OTPRepo,
		c.TemplateRenderer,
		c.MessageBroker,
		c.EmailSender,
		c.Logger,
	)

	c.ForgotPasswordService = services.NewForgotPasswordService(
		c.TemplateRenderer,
		c.MessageBroker,
		c.EmailSender,
		c.Logger,
	)

	strategies := map[entity.NotificationType][]ports.NotificationSender{
		entity.EmailNotification: []ports.NotificationSender{notification.NewEmailStrategy(
			c.EmailSender,
			c.Logger,
		)},
		entity.NotificationType(entity.InAppNotification): {notification.NewInAppStrategy(
			c.NotificationRepo,
			c.WSHub,
			c.Logger,
		)},
		// domain.SMSNotification: notification.NewSMSStrategy(...),
		// domain.PushNotification: notification.NewPushStrategy(...),
	}

	c.NotificationSender = notification.NewNotificationSender(strategies)

	// Notification Service
	c.NotificationService = services.NewNotificationService(
		c.NotificationRepo,
		c.NotificationSender,
		c.Logger,
	)

	return nil
}

func (c *Container) initEventHandlers() error {

	forgotPasswordEventHandler := events.NewForgotPasswordEventHandler(
		c.TemplateRenderer,
		c.NotificationRepo,
		c.MessageBroker,
		c.EmailSender,
		c.Logger,
	)
	otpRequestEventHandler := events.NewOTPRequestEventHandler(
		c.TemplateRenderer,
		c.NotificationRepo,
		c.OTPRepo,
		c.MessageBroker,
		c.EmailSender,
		c.Logger,
	)

	inAppNotificationChannelHandler := events.NewInAppNotificationChannelHandler(
		c.NotificationRepo,
		c.WSHub,
		c.Logger,
	)
	emailNotificationChannelHandler := events.NewHandleEmailNotificationChannel(
		c.NotificationRepo,
		c.EmailSender,
		c.Logger,
	)

	// Register handlers
	c.MessageBroker.Subscribe(string(domain_events.TopicAuthOTPRequested), otpRequestEventHandler)
	c.MessageBroker.Subscribe(string(domain_events.TopicNotificationRequestForgotPassword), forgotPasswordEventHandler)
	c.MessageBroker.Subscribe(string(domain_events.TopicNotificationInAppChannel), inAppNotificationChannelHandler)
	c.MessageBroker.Subscribe(string(domain_events.TopicNotificationEmailChannel), emailNotificationChannelHandler)

	return nil
}

func (c *Container) initPresentation() error {
	middlewareConfig := grpc_interface.MiddlewareConfig{
		ServiceName:    c.Config.ServiceName,
		ServiceVersion: c.Config.ServiceVersion,
		Environment:    c.Config.Environment,
		Logger:         c.Logger,
		Tracer:         c.Tracer,
	}
	grpcHandler := grpc_interface.NewHandler(
		c.NotificationService,
		c.EmailOTPService,
		c.ForgotPasswordService,
		c.Logger,
	)

	unaryInterceptors := grpc_interface.GetUnaryServerInterceptors(middlewareConfig)
	streamInterceptors := grpc_interface.GetStreamServerInterceptors(middlewareConfig)

	c.GRPCServer = grpc_server.NewServer(
		grpcHandler,
		c.Logger,
		grpc.UnaryInterceptor(unaryInterceptors),
		grpc.StreamInterceptor(streamInterceptors),
	)

	return nil
}

func (c *Container) Start(ctx context.Context) error {
	if err := c.KafkaConsumer.StartConsuming(ctx); err != nil {
		return fmt.Errorf("failed to start kafka consumer: %w", err)
	}

	go func() {
		if err := c.GRPCServer.Start(c.Config.GRPCAddress); err != nil {
			c.Logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	go func() {
		authFunc := ws.CreateAuthFunc(c.Config.JWTSecret, c.Logger)
		http.Handle("/notifications", c.WSHub.ServeWS(authFunc))

		addr := ":" + c.Config.WSPort
		c.Logger.Info("WebSocket server starting", zap.String("address", addr))

		if err := http.ListenAndServe(addr, nil); err != nil {
			c.Logger.Error("WebSocket server failed", zap.Error(err))
		}
	}()

	c.Logger.Info("All services started successfully")
	return nil
}

func (c *Container) Shutdown(ctx context.Context) error {
	c.Logger.Info("Shutting down services...")

	if err := c.WSHub.Shutdown(ctx); err != nil {
		c.Logger.Error("WebSocket hub shutdown error", zap.Error(err))
	}

	if err := c.KafkaConsumer.Close(); err != nil {
		c.Logger.Error("Kafka consumer shutdown error", zap.Error(err))
	}

	c.GRPCServer.Stop()

	if err := c.EmailSender.Close(); err != nil {
		c.Logger.Error("Email sender shutdown error", zap.Error(err))
	}

	if err := c.KafkaProducer.Close(); err != nil {
		c.Logger.Error("Kafka producer shutdown error", zap.Error(err))
	}

	if err := c.DB.Close(); err != nil {
		c.Logger.Error("Database shutdown error", zap.Error(err))
	}

	if c.Tracer != nil {
		if err := c.Tracer.Shutdown(ctx); err != nil {
			c.Logger.Error("Tracer shutdown error", zap.Error(err))
		}
	}

	c.Logger.Info("Shutdown complete")
	return nil
}

// // GetMetrics returns metrics from all components
// func (c *Container) GetMetrics() map[string]interface{} {
// 	return map[string]interface{}{
// 		"websocket":     c.WSHub.GetMetrics(),
// 		"kafka":         c.KafkaConsumer.GetMetrics(),
// 		"email_sender":  c.EmailSender.GetMetrics(),
// 		"grpc":          c.GRPCServer.GetMetrics(),
// 	}
// }
