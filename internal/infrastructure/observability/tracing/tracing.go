package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	log "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	// "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TracingConfig struct {
	CollectorEndpoint string

	ServiceName    string
	ServiceVersion string
	Environment    string
}

type Tracer struct {
	tracer trace.Tracer

	provider *sdktrace.TracerProvider

	logger ports.LoggerService
}

func NewTracer(config TracingConfig, logger ports.LoggerService) (*Tracer, error) {

	ctx := context.Background()

	conn, err := grpc.NewClient(
		config.CollectorEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect OTEL Collector: %w", err)
	}

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(config.CollectorEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	_ = conn

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", config.ServiceName),
			attribute.String("service.version", config.ServiceVersion),
			attribute.String("deployment.environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed creating resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(

		sdktrace.WithSampler(
			sdktrace.ParentBased(
				sdktrace.TraceIDRatioBased(1.0),
			),
		),

		sdktrace.WithBatcher(
			exporter,

			sdktrace.WithBatchTimeout(5*time.Second),

			sdktrace.WithExportTimeout(10*time.Second),

			sdktrace.WithMaxExportBatchSize(512),

			sdktrace.WithMaxQueueSize(2048),
		),

		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	tracer := provider.Tracer(config.ServiceName)

	logger.Info(
		"Tracing initialized",
		log.String("exporter", "otlp"),
		log.String("collector", config.CollectorEndpoint),
		log.String("service", config.ServiceName),
		log.String("version", config.ServiceVersion),
		log.String("environment", config.Environment),
	)

	return &Tracer{
		tracer:   tracer,
		provider: provider,
		logger:   logger,
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
