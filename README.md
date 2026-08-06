# Notification Service

The **Notification Service** is the communication service of the Edulearn platform. It is responsible for delivering transactional notifications, managing OTP workflows, sending password reset emails, and handling user notifications across multiple delivery channels.

The service is built with **Go**, **Hexagonal (Ports and Adapters) Architecture**, and **gRPC**, and integrates with the platform’s shared observability standards for structured logging, metrics, distributed tracing, and event-driven communication.

---

## Overview

The Notification Service is the authoritative owner of notification delivery and notification persistence within the platform. It consumes domain events from Kafka, renders templates, sends notifications through multiple channels, and exposes internal gRPC APIs for notification management.

### Responsibilities

* Transactional email delivery
* OTP generation and verification
* Password reset notification workflows
* In-app notification management
* Notification persistence
* Multi-channel notification processing
* Kafka event consumption
* Template rendering
* Rate limiting and delivery reliability

### Out of Scope

* User management (User Service)
* Authentication logic (Auth Service)
* Course management (Course Service)
* Payment processing (Payment Service)
* Order lifecycle management (Order Service)

---

# Architecture

This service follows **Hexagonal Architecture (Ports and Adapters)** with **SOLID principles**, enabling transport-independent business logic, high testability, and extensible notification channels.

## Layered Architecture

```text
             gRPC / WebSocket Interfaces
                      │
              Application Layer
     (Services / Events / Notification Workflows)
                      │
                 Domain Layer
   (Entities / Ports / Repository Interfaces / Events)
                      │
            Infrastructure Layer
(PostgreSQL / Redis / Kafka / SMTP / Templates / Observability)
```

### Layers

#### Presentation Layer

* gRPC handlers
* WebSocket hubs
* Request validation
* Transport-specific concerns

#### Application Layer

* Notification workflows
* OTP orchestration
* Password reset processing
* Channel routing
* Template rendering coordination
* Event handlers

#### Domain Layer

* Notification entity
* Message entity
* Notification events
* Repository interfaces
* Delivery channel ports
* Rate limiter ports

#### Infrastructure Layer

* PostgreSQL persistence
* Redis OTP storage
* Kafka consumer/producer
* SMTP email delivery
* Template rendering
* Logging, metrics, and tracing

---

# Technology Stack

| Category     | Technology                   |
| ------------ | ---------------------------- |
| Language     | Go 1.24+                     |
| Architecture | Hexagonal (Ports & Adapters) |
| Transport    | gRPC                         |
| Database     | PostgreSQL                   |
| ORM          | GORM                         |
| Cache        | Redis                        |
| Messaging    | Kafka                        |
| Email        | SMTP                         |
| Templates    | HTML Templates               |
| Logging      | Zap                          |
| Metrics      | Prometheus                   |
| Tracing      | OpenTelemetry                |
| Deployment   | Docker, Kubernetes, Helm     |

---

# Core Domain

The Notification Service owns the communication domain.

## Notification

* Notification metadata
* Delivery status
* Read/unread state
* Notification history
* User notification persistence

## OTP

* One-time password generation
* Verification
* Expiration
* Rate limiting
* Temporary storage

## Message

* Channel payload
* Delivery metadata
* Template variables
* Processing state

---

# Notification Channels

The service supports multiple delivery channels through a **Strategy Pattern** implementation.

```text
                NotificationSender
                        │
      ┌─────────────────┼─────────────────┐
      │                 │                 │
      ▼                 ▼                 ▼
Email Strategy     In-App Strategy    Future Channels
                                          │
                                   SMS / Push / WhatsApp
```

This design allows new notification channels to be added without modifying existing business workflows.

---

# OTP & Password Reset Flow

## OTP Request

```text
Auth Service
      │
      ▼
Kafka Event
      │
      ▼
Notification Service
      │
      ▼
Generate OTP
      │
      ▼
Store in Redis
      │
      ▼
Render Email Template
      │
      ▼
SMTP Delivery
```

## Password Reset

```text
Forgot Password Request
          │
          ▼
Generate Reset Context
          │
          ▼
Render Email Template
          │
          ▼
SMTP Delivery
          │
          ▼
User Receives Reset Link
```

---

# Project Structure

```text
cmd/
└── server/

internal/
├── application/
│   ├── events/
│   ├── services/
│   ├── ports/
│   └── interfaces/
├── domain/
│   ├── entities/
│   ├── events/
│   ├── repositories/
│   └── errors/
├── infrastructure/
│   ├── database/
│   ├── kafka/
│   ├── redis/
│   ├── email/
│   ├── template/
│   ├── notification/
│   └── observability/
└── presentation/
    ├── grpc/
    └── websocket/

pkg/
└── templates/
```

---

# Communication

## gRPC APIs

The Notification Service exposes internal gRPC APIs consumed by:

* API Gateway
* Auth Service
* User Service
* Course Service
* Payment Service
* Order Service
* Chat Service

Example operations:

* SendEmail
* SendOTP
* VerifyOTP
* CreateNotification
* GetNotifications
* MarkAsRead
* DeleteNotification
* SendTemplateNotification

---

## Kafka Integration

The Notification Service is primarily **event-driven** and consumes notification requests from Kafka.

### Consumed Events

| Topic                                        | Purpose                    |
| -------------------------------------------- | -------------------------- |
| notification.request.auth.otp.v1             | Send authentication OTP    |
| notification.request.auth.forgot-password.v1 | Send password reset email  |
| notification.channel.email.v1                | Send transactional email   |
| notification.channel.inapp.v1                | Create in-app notification |
| course.published.v1                          | Notify interested users    |
| enrollment.created.v1                        | Enrollment confirmation    |
| payment.completed.v1                         | Payment confirmation       |

### Published Events

| Topic                        | Purpose                |
| ---------------------------- | ---------------------- |
| notification.email.sent.v1   | Email delivered        |
| notification.email.failed.v1 | Email delivery failed  |
| notification.otp.sent.v1     | OTP delivered          |
| notification.otp.verified.v1 | OTP verified           |
| notification.created.v1      | Notification persisted |

This asynchronous architecture decouples notification delivery from business services and improves overall system resilience.

---

# Data Ownership

The Notification Service is the single source of truth for notification-related data.

| Entity                  | Owner                |
| ----------------------- | -------------------- |
| notifications           | Notification Service |
| processed_notifications | Notification Service |
| otp_records (Redis)     | Notification Service |

Other services interact with this data through gRPC APIs or Kafka events rather than direct database access.

---

# Observability

The service follows the platform-wide observability architecture based on **OpenTelemetry**, **Prometheus**, **Grafana**, **Loki**, and **Tempo**.

## Logging

* Structured JSON logs
* Zap logger
* Correlation IDs
* Trace-aware logging
* Delivery diagnostics

## Metrics

Prometheus metrics include:

* Email send requests
* Email delivery success/failure
* OTP generation rate
* OTP verification rate
* Notification processing latency
* Kafka consumer lag
* SMTP delivery duration
* Template rendering duration

Exposed at:

```text
/metrics
```

## Distributed Tracing

OpenTelemetry instrumentation provides end-to-end tracing across notification workflows.

Trace flow:

```text
Auth Service
      │
      ▼
Notification Service
      │
      ▼
Redis / SMTP / PostgreSQL / Kafka
```

Traces are exported to **OTEL Collector → Tempo → Grafana**.

---

# Redis Usage

Redis is used for:

* OTP storage
* OTP expiration
* Verification state
* Rate limiting
* Temporary notification metadata
* Duplicate delivery prevention

---

# Database

PostgreSQL is the primary persistent datastore.

GORM manages:

* Entity mapping
* Migrations
* Repository implementations
* Transaction management

Typical migration command:

```bash
go run ./cmd/server migrate
```

---

# Local Development

## Prerequisites

* Go 1.24+
* PostgreSQL
* Redis
* Kafka
* SMTP server

## Install Dependencies

```bash
go mod download
```

## Run

```bash
go run ./cmd/server
```

## Build

```bash
go build -o notification-service ./cmd/server
```

## Run Binary

```bash
./notification-service
```

---

# Environment Variables

| Variable                    | Description                  |
| --------------------------- | ---------------------------- |
| PORT                        | gRPC server port             |
| DATABASE_URL                | PostgreSQL connection string |
| REDIS_URL                   | Redis connection string      |
| KAFKA_BROKERS               | Kafka broker list            |
| SMTP_HOST                   | SMTP server host             |
| SMTP_PORT                   | SMTP server port             |
| SMTP_USERNAME               | SMTP username                |
| SMTP_PASSWORD               | SMTP password                |
| SMTP_FROM                   | Default sender address       |
| OTEL_EXPORTER_OTLP_ENDPOINT | OTLP collector endpoint      |
| LOG_LEVEL                   | Logging level                |

See `env.example` for the complete configuration.

---

# Docker

The service uses a **multi-stage Docker build** optimized for production.

Optimizations include:

* Multi-stage compilation
* Static binary generation
* Minimal runtime image
* Non-root execution
* Small attack surface
* Fast startup time

---

# Kubernetes Deployment

Deployment is managed through the **Edulearn umbrella Helm chart**.

The service is deployed with:

* ClusterIP service
* gRPC exposure
* Liveness probes
* Readiness probes
* Resource requests and limits
* Horizontal Pod Autoscaler support
* Prometheus ServiceMonitor

---

# CI/CD

This service participates in the platform GitOps deployment pipeline.

```text
Git Push
    │
    ▼
GitHub Actions
    ├── Test
    ├── Build
    ├── Lint
    ├── Trivy Scan
    └── Push to GHCR
             │
             ▼
ArgoCD Image Updater
             │
             ▼
ArgoCD
             │
             ▼
Amazon EKS
```

---

# Performance Optimizations

Implemented optimizations include:

* Redis-backed OTP storage
* SMTP connection reuse
* Kafka asynchronous processing
* Template caching
* Batched notification processing
* Connection pooling
* Efficient GORM queries
* Optimized Docker image size

---

# Security

The service follows production-oriented security practices.

## Notification Security

* OTP expiration enforcement
* Rate limiting
* Duplicate delivery prevention
* Secure token handling
* Template input sanitization

## Secrets Management

Production deployments retrieve secrets from:

* AWS Secrets Manager
* External Secrets Operator

## Container Security

* Runs as non-root user
* No shell access
* Minimal Linux capabilities
* Read-only filesystem where applicable

---

# Testing

```bash
# Unit tests
go test ./test/unit/...

# Integration tests
go test ./test/integration/...

# End-to-end tests
go test ./test/e2e/...

# Coverage
go test ./... -cover
```

---


# Related Repositories

| Repository                    | Description                                                   |
| ----------------------------- | ------------------------------------------------------------- |
| [edulearn-platform](https://github.com/muhammed-shafeeque-th/edulearn-platform)             | Platform orchestration repository                             |
| [edulearn-api-gateway](https://github.com/muhammed-shafeeque-th/edulearn-api-gateawy)          | API Gateway                                                   |
| [edulearn-user-service](https://github.com/muhammed-shafeeque-th/edulearn-user-srv)         | User profile service                                          |
| [edulearn-course-service](https://github.com/muhammed-shafeeque-th/edulearn-course-srv)       | Course management service                                     |
| [edulearn-payment-service](https://github.com/muhammed-shafeeque-th/edulearn-payment-srv)      | Payment processing service                                    |
| [edulearn-auth-service](https://github.com/muhammed-shafeeque-th/edulearn-auth-srv)      | Authentication service                                    |
| [edulearn-client](https://github.com/muhammed-shafeeque-th/edulearn-client)        | Edulearn Frontend                                      |
| [edulearn-chat-service](https://github.com/muhammed-shafeeque-th/edulearn-chat-srv) | Chat service                                          |
| [edulearn-auth-service](https://github.com/muhammed-shafeeque-th/edulearn-auth-srv)         | Authentication service                                        |
| [@edulearn/core](https://github.com/muhammed-shafeeque-th/edulearn-core)                | Shared logging, metrics, tracing, Redis, Kafka, health checks |
| [@edulearn/nest](https://github.com/muhammed-shafeeque-th/edulearn-nest)                | Shared NestJS infrastructure package                          |

---

# License

This project is part of the **Edulearn Platform** and is licensed under the MIT License.
