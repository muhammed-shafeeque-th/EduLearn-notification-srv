package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	EmailSentTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_server_email_sent_total",
			Help: "Total number of email sent",
		},
	)

	KafkaMessageProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_kafka_message_processed_total",
			Help: "Total number of Kafka messages processed",
		},
	)
	OTPSentTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_otp_send_total",
			Help: "Total number of OTP send ",
		},
	)
	EmailSendErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_email_send_errors_total",
			Help: "Total number of email send errors",
		},
	)
	EmailRetries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_email_retries",
			Help: "Total number of retries for email sent",
		},
	)
	EmailRateLimited = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_service_email_rate_limited",
			Help: "Total number of rate limits for email sent",
		},
	)
)

func InitMetrics() {
	prometheus.MustRegister(
		EmailSentTotal,
		EmailSendErrors,
		KafkaMessageProcessed,
		OTPSentTotal)
}

func StartMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		http.ListenAndServe(":9090", nil)
	}()
}
func GetHandler() http.Handler {
	return promhttp.Handler()
}
