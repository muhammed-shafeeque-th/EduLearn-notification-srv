# Notification Service

The **Notification Service** handles email notifications, OTP management, password reset workflows, and notification management in the EduLearn platform. It is built with **Go**, uses **gRPC** for inter-service communication, **Kafka** for async message processing, **Redis** for OTP storage, and **PostgreSQL** with **GORM** for persistence.

## 📚 Documentation

This service documentation is organized into several focused documents:

- **[Overview](./docs/overview.md)** - Service purpose, scope, responsibilities, and key features
- **[Architecture](./docs/architecture.md)** - Internal design, layers, patterns, and technical decisions
- **[API Reference](./docs/api.md)** - Complete gRPC service definitions
- **[Database](./docs/database.md)** - Entity models, relationships, and data ownership
- **[Events](./docs/events.md)** - Kafka events consumed and processed
- **[Flows](./docs/flows.md)** - Notification and OTP flows
- **[Errors](./docs/errors.md)** - Error handling and codes

## 🚀 Quick Start

### Prerequisites

- **Go** (v1.24.1 or higher)
- **PostgreSQL** (v13+)
- **Redis** (v6+)
- **Kafka** (v2.8+)

### Installation

```bash
# Clone repository
git clone <repository-url>
cd notification

# Install dependencies
go mod download

# Copy environment file
cp env.example .env
# Edit .env with your configuration
```

### Running the Service

```bash
# Development
go run cmd/server/main.go

# Build
go build -o notification cmd/server/main.go
./notification
```

## 📋 Key Features

- **Email Notifications**: Send transactional emails via SMTP
- **OTP Management**: Generate and verify OTPs
- **Password Reset**: Handle password reset workflows
- **Notification Management**: CRUD operations for user notifications
- **Kafka Integration**: Consume notification requests from Kafka
- **Observability**: OpenTelemetry tracing, Prometheus metrics, structured logging

## 📄 License

MIT License
