package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const requestDurationMetricName = "http.server.request.duration.milliseconds"

// Metrics records request duration after the complete Gin handler chain has run.
// Route and status are bounded dimensions; raw URLs and request data are not
// recorded to avoid high-cardinality metric series.
func Metrics() gin.HandlerFunc {
	duration, err := otel.Meter("ledger-service/http").Float64Histogram(
		requestDurationMetricName,
		metric.WithDescription("HTTP server request duration in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}
		duration.Record(c.Request.Context(), time.Since(startedAt).Seconds()*1000,
			metric.WithAttributes(
				attribute.String("http.request.method", c.Request.Method),
				attribute.String("http.route", route),
				attribute.String("http.response.status_code", strconv.Itoa(c.Writer.Status())),
			),
		)
	}
}
