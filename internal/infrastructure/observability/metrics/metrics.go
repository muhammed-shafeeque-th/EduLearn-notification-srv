package metrics

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsConfig struct {
	Port string
	Path string
}

type MetricsService struct {
	registry *prometheus.Registry
	server   *http.Server
	once     sync.Once
}

var (

	EmailSentTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_email_sent_total",
			Help: "Total number of emails sent",
		},
	)

	KafkaMessageProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_kafka_messages_processed_total",
			Help: "Total number of Kafka messages processed",
		},
	)

	OTPSentTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_otp_sent_total",
			Help: "Total number of OTPs sent",
		},
	)

	EmailSendErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_email_send_errors_total",
			Help: "Total number of email send failures",
		},
	)

	EmailRetries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_email_retries_total",
			Help: "Total number of email retries",
		},
	)

	EmailRateLimited = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_email_rate_limited_total",
			Help: "Total number of rate limited email attempts",
		},
	)
)

func NewMetricsService() *MetricsService {
	return &MetricsService{
		registry: prometheus.NewRegistry(),
	}
}

func (m *MetricsService) Initialize(config MetricsConfig) error {

	var err error

	m.once.Do(func() {

		err = m.registry.Register(EmailSentTotal)
		if err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return
		}

		err = m.registry.Register(EmailSendErrors)
		if err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return
		}

		err = m.registry.Register(KafkaMessageProcessed)
		if err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return
		}

		err = m.registry.Register(OTPSentTotal)
		if err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return
		}

		err = m.registry.Register(EmailRetries)
		if err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return
		}

		err = m.registry.Register(EmailRateLimited)
		if err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return
		}

		// mux := http.NewServeMux()
		// mux.Handle(config.Path, promhttp.HandlerFor(
		// 	m.registry,
		// 	promhttp.HandlerOpts{},
		// ))

		// m.server = &http.Server{
		// 	Addr:              ":" + config.Port,
		// 	Handler:           mux,
		// 	ReadHeaderTimeout: 5 * time.Second,
		// }

		// go func() {
		// 	_ = m.server.ListenAndServe()
		// }()
	})

	return err
}

func (m *MetricsService) Shutdown(ctx context.Context) error {

	if m.server == nil {
		return nil
	}

	return m.server.Shutdown(ctx)
}

func (m *MetricsService) Handler() http.Handler {
	return promhttp.HandlerFor(
		m.registry,
		promhttp.HandlerOpts{},
	)
}