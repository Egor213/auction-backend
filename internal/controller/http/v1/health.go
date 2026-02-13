package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type healthRoutes struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func newHealthRoutes(e *echo.Echo, db *pgxpool.Pool, rdb *redis.Client) {
	r := &healthRoutes{db: db, redis: rdb}
	e.GET("/health", r.health)
	e.GET("/ready", r.ready)
}

func (r *healthRoutes) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (r *healthRoutes) ready(c echo.Context) error {
	checks := map[string]string{}
	allOk := true

	if err := r.db.Ping(c.Request().Context()); err != nil {
		checks["postgres"] = "error: " + err.Error()
		allOk = false
	} else {
		checks["postgres"] = "ok"
	}

	if err := r.redis.Ping(c.Request().Context()).Err(); err != nil {
		checks["redis"] = "error: " + err.Error()
		allOk = false
	} else {
		checks["redis"] = "ok"
	}

	status := http.StatusOK
	if !allOk {
		status = http.StatusServiceUnavailable
	}

	return c.JSON(status, map[string]any{
		"checks": checks,
	})
}
