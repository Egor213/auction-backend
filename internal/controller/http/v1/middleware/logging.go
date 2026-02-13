package mw

import (
	"time"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

func LoggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			log.WithFields(log.Fields{
				"method":    c.Request().Method,
				"path":      c.Request().URL.Path,
				"status":    c.Response().Status,
				"latency":   time.Since(start).String(),
				"remote_ip": c.RealIP(),
			}).Info("HTTP request")

			return err
		}
	}
}
