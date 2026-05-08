package observability

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "messenger_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// WebSocket metrics
	wsConnectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_websocket_connections_total",
			Help: "Total number of WebSocket connections",
		},
	)

	wsMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"type"},
	)

	// Auth metrics
	authAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"method", "success"},
	)

	// Message metrics
	messagesSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_messages_sent_total",
			Help: "Total number of messages sent",
		},
		[]string{"type", "chat_type"},
	)

	messageLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "messenger_message_latency_seconds",
			Help:    "Message delivery latency in seconds",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
	)
)

var tracer trace.Tracer

// Init initializes observability components
func Init(logger *zap.Logger) {
	tracer = otel.Tracer("messenger-server")
	logger.Info("Observability initialized")
}

// Middleware returns a Gin middleware for observability
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		
		// Process request
		c.Next()
		
		// Record metrics
		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		
		httpRequestsTotal.WithLabelValues(c.Request.Method, path, string(rune(status))).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
		
		// Log slow requests
		if duration > 1.0 {
			zap.L().Warn("Slow request detected",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Float64("duration", duration),
				zap.Int("status", status),
			)
		}
	}
}

// MetricsHandler returns the Prometheus metrics handler
func MetricsHandler() gin.HandlerFunc {
	handler := promhttp.Handler()
	
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(method string, success bool) {
	successStr := "false"
	if success {
		successStr = "true"
	}
	authAttemptsTotal.WithLabelValues(method, successStr).Inc()
}

// RecordMessageSent records a sent message
func RecordMessageSent(msgType, chatType string, latency time.Duration) {
	messagesSentTotal.WithLabelValues(msgType, chatType).Inc()
	messageLatency.Observe(latency.Seconds())
}

// RecordWSConnection records a WebSocket connection
func RecordWSConnection() {
	wsConnectionsTotal.Inc()
}

// RecordWSMessage records a WebSocket message
func RecordWSMessage(msgType string) {
	wsMessagesTotal.WithLabelValues(msgType).Inc()
}

// StartSpan starts a new tracing span
func StartSpan(c *gin.Context, name string) (trace.Span, *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), name)
	c.Request = c.Request.WithContext(ctx)
	return span, c
}

// HealthCheckWithMetrics returns health status with metrics
func HealthCheckWithMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "healthy",
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"version":     "0.1.0-mvp",
			"metrics_path": "/metrics",
		})
	}
}
