# Notification Service - Overview

## Purpose

The Notification Service handles email notifications, OTP management, password reset workflows, and notification management in the EduLearn platform. It provides reliable notification delivery and management.

## Scope & Responsibilities

### Core Responsibilities

1. **Email Notifications**
   - Send transactional emails
   - Template-based email rendering
   - SMTP integration

2. **OTP Management**
   - Generate OTPs
   - Verify OTPs
   - OTP expiration handling
   - Redis-backed OTP storage

3. **Password Reset**
   - Password reset email sending
   - Reset token management

4. **Notification Management**
   - Create notifications
   - Retrieve user notifications
   - Mark notifications as read
   - Delete notifications
   - Notification filtering

5. **Event Consumption**
   - Consumes notification requests from Kafka
   - Processes notification events
   - Multi-channel notification support

### Out of Scope

- User management (User Service)
- Authentication (Auth Service)
- Business logic for other domains

## Folder Structure
```
notification/
├── cmd/                         # Application entry points
│   └── server/                  # Main server
│       └── main.go              # Application bootstrap
├── internal/                    # Private application code
│   ├── application/             # Application layer
│   │   ├── events/              # Event handlers
│   │   │   ├── handle_email_notification_channel.go # Email channel handler
│   │   │   ├── handle_forgot_password_request.go     # Forgot password handler
│   │   │   ├── handle_inapp_notification_channel.go  # In-app channel handler
│   │   │   └── handle_send_otp_request.go            # OTP request handler
│   │   ├── interfaces/          # Application interfaces
│   │   │   ├── cache_interface.go        # Cache interface
│   │   │   ├── kafka_producer_interface.go # Kafka producer interface
│   │   │   └── sender_strategy_interface.go # Sender strategy interface
│   │   ├── ports/               # Port interfaces (Hexagonal Architecture)
│   │   │   ├── cache_service.go             # Cache service port
│   │   │   ├── email_sender.go              # Email sender port
│   │   │   ├── message_brocker.go           # Message broker port
│   │   │   ├── notification_processor.go    # Notification processor port
│   │   │   ├── processed_notification_repository.go # Repository port
│   │   │   ├── rate_limiter.go              # Rate limiter port
│   │   │   └── sender_strategy.go           # Sender strategy port
│   │   └── services/            # Application services
│   │       ├── email_otp_service.go         # Email OTP service
│   │       ├── forgot_password_service.go   # Forgot password service
│   │       └── notification_service.go      # Notification service
│   ├── bootstrap/               # Application bootstrap
│   │   └── di.go                # Dependency injection setup
│   ├── domain/                  # Domain layer
│   │   ├── entities/            # Domain entities
│   │   │   ├── message.go       # Message entity
│   │   │   └── notification.go  # Notification entity
│   │   ├── errors/              # Domain errors
│   │   │   └── errors.go        # Custom error types
│   │   ├── events/              # Domain events
│   │   │   ├── domin_request_events.go  # Domain request events
│   │   │   ├── event_topics.go          # Event topic definitions
│   │   │   └── events.go                # Event structures
│   │   └── repositories/        # Repository interfaces
│   │       └── notification_repository.go # Notification repository interface
│   ├── infrastructure/          # Infrastructure layer
│   │   ├── config/              # Configuration
│   │   │   └── config.go        # Application configuration
│   │   ├── database/            # Database implementation
│   │   │   └── gorm/            # GORM implementation
│   │   │       ├── connection.go    # Database connection
│   │   │       ├── migrations/      # Database migrations
│   │   │       └── repositories/    # Repository implementations
│   │   ├── email/               # Email infrastructure
│   │   │   └── email_sender.go  # Email sending implementation
│   │   ├── errors/              # Infrastructure errors
│   │   │   └── errors.go        # Error handling
│   │   ├── grpc/                # gRPC implementation
│   │   │   └── server.go        # gRPC server setup
│   │   ├── health/              # Health checks
│   │   │   └── health.go        # Health check implementation
│   │   ├── kafka/               # Kafka implementation
│   │   │   ├── consumer.go      # Kafka consumer
│   │   │   ├── producer.go      # Kafka producer
│   │   │   └── topics.go        # Topic definitions
│   │   ├── metrics/             # Metrics collection
│   │   │   └── metrics.go       # Prometheus metrics
│   │   ├── notification/        # Notification infrastructure
│   │   │   ├── email_strategy.go    # Email sending strategy
│   │   │   ├── inapp_strategy.go    # In-app notification strategy
│   │   │   └── sender.go            # Notification sender
│   │   ├── observability/       # Monitoring
│   │   │   ├── logging/         # Logging setup
│   │   │   ├── metrics/         # Metrics collection
│   │   │   └── tracing/         # Distributed tracing
│   │   ├── ratelimit/           # Rate limiting
│   │   │   └── retelimit.go     # Rate limiter implementation
│   │   ├── redis/               # Redis implementation
│   │   │   └── redis.go         # Redis service
│   │   └── template/            # Template rendering
│   │       └── renderer.go      # Template renderer
│   ├── interfaces/              # Interface definitions
│   │   ├── grpc/                # gRPC interfaces
│   │   │   └── handler.go       # gRPC handler interface
│   │   └── websocket/           # WebSocket interfaces
│   │       └── hub.go           # WebSocket hub interface
│   └── presentation/            # Presentation layer
│       ├── grpc/                # gRPC handlers
│       │   ├── handler.go       # gRPC request handlers
│       │   └── server.go        # gRPC server setup
│       └── websocket/           # WebSocket implementation
│           └── hub.go           # WebSocket hub
├── pkg/                         # Public packages
│   ├── templates/               # Email templates
│   │   ├── activation-mail.html # Account activation template
│   │   ├── activation-mail1.html # Alternative activation template
│   │   └── forgot-mail.html     # Password reset template
│   └── utils/                   # Utility functions
│       └── generate_uuid.go     # UUID generation utility
├── proto/                       # Protocol buffer definitions
│   ├── notification.proto       # Notification service protobuf
│   └── proto/                   # Generated protobuf code
│       ├── notification_grpc.pb.go # Generated gRPC code
│       └── notification.pb.go   # Generated protobuf types
├── test/                        # Test files
│   ├── e2e/                     # End-to-end tests
│   │   └── notification_e2e_test.go # E2E test suite
│   ├── integration/             # Integration tests
│   │   └── notification_repository_integration_test.go # Repository tests
│   └── unit/                    # Unit tests
│       ├── mocks/               # Mock implementations
│       └── service/             # Service unit tests
├── config.yaml                  # Application configuration
├── Dockerfile                   # Docker configuration
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
├── env.example                  # Environment variables template
├── LICENSE                      # License
├── README.md                    # Service documentation
├── main.exe                     # Compiled binary (Windows)
└── logs/                        # Application logs
```

## Key Features

- **Email Delivery**: Reliable email sending via SMTP
- **OTP Management**: Secure OTP generation and verification
- **Template Rendering**: Handlebars template support
- **Notification Persistence**: Store and retrieve notifications
- **Event-Driven**: Kafka-based notification requests
- **Multi-Channel**: Email, SMS, Push, In-App support

## Service Boundaries

### Owns Data For

- Notifications
- OTP records (Redis)
- Processed notifications

### Depends On

- **SMTP Server**: Email delivery
- **Database**: PostgreSQL for notification persistence
- **Redis**: OTP storage
- **Kafka**: Notification request consumption

## Technical Stack

- **Language**: Go 1.24+
- **Framework**: Custom framework
- **Database**: PostgreSQL with GORM
- **Cache**: Redis
- **Messaging**: Kafka
- **RPC**: gRPC
- **Email**: SMTP
- **Templates**: Handlebars

## Key Entities

- **Notification**: User notifications
- **OTP**: One-time password records
- **ProcessedNotification**: Processed notification tracking

