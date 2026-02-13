package httpapi

import (
	mw "auction-platform/internal/controller/http/v1/middleware"
	"auction-platform/internal/metrics"
	"auction-platform/internal/service"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func ConfigureRouter(
	handler *echo.Echo,
	services *service.Services,
	m *metrics.Metrics,
	pool *pgxpool.Pool,
	rdb *redis.Client,
	rps float64,
	burst int,
) {
	handler.Use(middleware.Recover())
	handler.Use(mw.LoggingMiddleware())
	handler.Use(mw.MetricsMiddleware(m))

	rl := mw.NewRateLimiter(rps, burst, m)
	handler.Use(rl.Middleware())

	handler.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	newHealthRoutes(handler, pool, rdb)

	api := handler.Group("/api/v1")
	{
		newAuctionRoutes(api.Group("/auction"), services.Auctions)
		newBidRoutes(api.Group("/bid"), services.Bids)
	}

	handler.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "auction platform")
	})
}
