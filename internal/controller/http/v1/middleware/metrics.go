package mw

import (
	"auction-platform/internal/metrics"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func MetricsMiddleware(m *metrics.Metrics) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			m.HTTPActiveRequests.Inc()

			err := next(c)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(c.Response().Status)
			method := c.Request().Method
			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}

			m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
			m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
			m.HTTPActiveRequests.Dec()

			return err
		}
	}
}
