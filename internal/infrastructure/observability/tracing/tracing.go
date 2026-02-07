package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type TracingConfig struct {
	JaegerHost     string
	JaegerPort     string
	ServiceName    string
	ServiceVersion string
	Environment    string
}

type Tracer struct {
	tracer trace.Tracer
	logger *zap.Logger
}

func NewTracer(config TracingConfig, logger *zap.Logger) (*Tracer, error) {
	// Create Jaeger exporter
	jaegerEndpoint := fmt.Sprintf("http://%s:%s/api/traces", config.JaegerHost, config.JaegerPort)
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(100),
		),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Create tracer
	tracer := tp.Tracer(config.ServiceName)

	logger.Info("Tracing initialized",
		zap.String("jaeger_endpoint", jaegerEndpoint),
		zap.String("service_name", config.ServiceName),
		zap.String("service_version", config.ServiceVersion),
	)

	return &Tracer{
		tracer: tracer,
		logger: logger,
	}, nil
}

func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

func (t *Tracer) Shutdown(ctx context.Context) error {
	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		return tp.Shutdown(ctx)
	}
	return nil
}

// Helper function to add common span attributes
func AddSpanAttributes(span trace.Span, attributes map[string]string) {
	for key, value := range attributes {
		span.SetAttributes(attribute.String(key, value))
	}
}

// Helper function to record error in span
func RecordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
