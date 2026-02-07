package grpc_interface

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/errors"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/observability/logging"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// MiddlewareConfig holds configuration for gRPC middleware
type MiddlewareConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Logger         *zap.Logger
	Tracer         *tracing.Tracer
}

// CreateUnaryServerInterceptors creates all unary server interceptors
func CreateUnaryServerInterceptors(config MiddlewareConfig) []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{
		// Recovery interceptor (should be first)
		recovery.UnaryServerInterceptor(
			recovery.WithRecoveryHandler(recoveryHandler(config.Logger)),
		),

		// Tracing interceptor
		tracingUnaryInterceptor(config),

		// Logging interceptor
		loggingUnaryInterceptor(config),

		// Metrics interceptor
		grpc_prometheus.UnaryServerInterceptor,

		// // Error handling interceptor
		// errorHandlingUnaryInterceptor(config),

		// // Request ID interceptor
		// requestIDUnaryInterceptor(),
	}
}

// CreateStreamServerInterceptors creates all stream server interceptors
func CreateStreamServerInterceptors(config MiddlewareConfig) []grpc.StreamServerInterceptor {
	return []grpc.StreamServerInterceptor{
		recovery.StreamServerInterceptor(
			recovery.WithRecoveryHandler(recoveryHandler(config.Logger)),
		),

		// Tracing interceptor
		tracingStreamInterceptor(config),

		// Logging interceptor
		loggingStreamInterceptor(config),

		// Metrics interceptor
		grpc_prometheus.StreamServerInterceptor,

		// // Error handling interceptor
		// errorHandlingStreamInterceptor(config),

		// // Request ID interceptor
		// requestIDStreamInterceptor(),
	}
}

func GetStreamServerInterceptors(config MiddlewareConfig) grpc.StreamServerInterceptor {
	streamsInterceptors := CreateStreamServerInterceptors(config)

	return chainStreamInterceptors(streamsInterceptors)

}
func GetUnaryServerInterceptors(config MiddlewareConfig) grpc.UnaryServerInterceptor {
	unaryInterceptors := CreateUnaryServerInterceptors(config)

	return chainUnaryInterceptors(unaryInterceptors)

}

func chainUnaryInterceptors(interceptors []grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if len(interceptors) == 0 {
			return handler(ctx, req)
		}

		currentHandler := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			nextHandler := currentHandler
			currentHandler = func(ctx context.Context, req interface{}) (interface{}, error) {
				return interceptor(ctx, req, info, nextHandler)
			}
		}
		return currentHandler(ctx, req)
	}
}

func chainStreamInterceptors(interceptors []grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if len(interceptors) == 0 {
			return handler(srv, ss)
		}

		currentHandler := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			nextHandler := currentHandler
			currentHandler = func(srv interface{}, ss grpc.ServerStream) error {
				return interceptor(srv, ss, info, nextHandler)
			}
		}
		return currentHandler(srv, ss)
	}
}

// Recovery handler for panics
func recoveryHandler(logger *zap.Logger) recovery.RecoveryHandlerFunc {
	return func(p interface{}) error {
		logger.Error("Panic recovered in gRPC handler",
			zap.Any("panic", p),
			zap.String("stack", getStackTrace()),
		)
		return status.Errorf(codes.Internal, "Internal server error")
	}
}

// Tracing interceptor for unary calls
func tracingUnaryInterceptor(config MiddlewareConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract trace context from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Start span
		spanCtx, span := config.Tracer.StartSpan(ctx, info.FullMethod)
		defer span.End()

		// Add span attributes
		span.SetAttributes(
			attribute.String("grpc.method", info.FullMethod),
			attribute.String("grpc.service", config.ServiceName),
			attribute.String("grpc.version", config.ServiceVersion),
			attribute.String("environment", config.Environment),
		)

		// Add request metadata as span attributes
		if userAgent := getMetadataValue(md, "user-agent"); userAgent != "" {
			span.SetAttributes(attribute.String("user_agent", userAgent))
		}

		// Execute handler with span context
		resp, err := handler(spanCtx, req)

		// Record error if any
		if err != nil {
			tracing.RecordError(span, err)
		}

		return resp, err
	}
}

// Tracing interceptor for stream calls
func tracingStreamInterceptor(config MiddlewareConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Extract trace context from metadata
		// md, ok := metadata.FromIncomingContext(ss.Context())
		// if !ok {
		// 	md = metadata.New(nil)
		// }

		// Start span
		spanCtx, span := config.Tracer.StartSpan(ss.Context(), info.FullMethod)
		defer span.End()

		// Add span attributes
		span.SetAttributes(
			attribute.String("grpc.method", info.FullMethod),
			attribute.String("grpc.service", config.ServiceName),
			attribute.String("grpc.version", config.ServiceVersion),
			attribute.String("environment", config.Environment),
			attribute.Bool("grpc.stream", true),
		)

		// Create wrapped stream with span context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          spanCtx,
		}

		// Execute handler
		err := handler(srv, wrappedStream)

		// Record error if any
		if err != nil {
			tracing.RecordError(span, err)
		}

		return err
	}
}

// Logging interceptor for unary calls
func loggingUnaryInterceptor(config MiddlewareConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Extract request ID
		requestID := getRequestID(ctx)

		// Create request logger
		reqLogger := logging.NewRequestLogger(
			config.Logger,
			"gRPC",
			info.FullMethod,
			getUserID(ctx),
		).With(
			zap.String("request_id", requestID),
			zap.String("grpc_method", info.FullMethod),
		)

		reqLogger.Info("gRPC request started")

		// Execute handler
		resp, err := handler(ctx, req)

		// Log response
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Internal
			}
		}

		reqLogger.Info("gRPC request completed",
			zap.Duration("duration", duration),
			zap.String("status", statusCode.String()),
			zap.Error(err),
		)

		return resp, err
	}
}

// Logging interceptor for stream calls
func loggingStreamInterceptor(config MiddlewareConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Extract request ID
		requestID := getRequestID(ss.Context())

		// Create request logger
		reqLogger := logging.NewRequestLogger(
			config.Logger,
			"gRPC-Stream",
			info.FullMethod,
			getUserID(ss.Context()),
		).With(
			zap.String("request_id", requestID),
			zap.String("grpc_method", info.FullMethod),
		)

		reqLogger.Info("gRPC stream started")

		// Execute handler
		err := handler(srv, ss)

		// Log response
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Internal
			}
		}

		reqLogger.Info("gRPC stream completed",
			zap.Duration("duration", duration),
			zap.String("status", statusCode.String()),
			zap.Error(err),
		)

		return err
	}
}

// Error handling interceptor for unary calls
func errorHandlingUnaryInterceptor(config MiddlewareConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)

		if err != nil {
			// Convert error to structured error
			serviceErr := errors.FromError(err)

			// Log error with context
			errors.LogError(ctx, config.Logger, serviceErr,
				zap.String("grpc_method", info.FullMethod),
				zap.String("request_id", getRequestID(ctx)),
			)

			// Convert to gRPC status
			return resp, errors.ToGRPCStatus(serviceErr).Err()
		}

		return resp, nil
	}
}

// Error handling interceptor for stream calls
func errorHandlingStreamInterceptor(config MiddlewareConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)

		if err != nil {
			// Convert error to structured error
			serviceErr := errors.FromError(err)

			// Log error with context
			errors.LogError(ss.Context(), config.Logger, serviceErr,
				zap.String("grpc_method", info.FullMethod),
				zap.String("request_id", getRequestID(ss.Context())),
			)

			// Convert to gRPC status
			return errors.ToGRPCStatus(serviceErr).Err()
		}

		return nil
	}
}

// Request ID interceptor for unary calls
func requestIDUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := getRequestID(ctx)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to context
		newCtx := context.WithValue(ctx, "request_id", requestID)

		return handler(newCtx, req)
	}
}

// Request ID interceptor for stream calls
func requestIDStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		requestID := getRequestID(ss.Context())
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Create wrapped stream with request ID
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), "request_id", requestID),
		}

		return handler(srv, wrappedStream)
	}
}

// Helper functions
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}

	// Try to get from metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-request-id"); len(values) > 0 {
			return values[0]
		}
	}

	return ""
}

func getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}

	// Try to get from metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-user-id"); len(values) > 0 {
			return values[0]
		}
	}

	return ""
}

func getMetadataValue(md metadata.MD, key string) string {
	if values := md.Get(key); len(values) > 0 {
		return values[0]
	}
	return ""
}

func getStackTrace() string {
	// This is a simplified version - in production you might want to use runtime/debug
	return "stack trace not implemented"
}

// Wrapped server stream for context propagation
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
