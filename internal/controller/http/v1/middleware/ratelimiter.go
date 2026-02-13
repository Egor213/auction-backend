package mw

import (
	hd "auction-platform/internal/controller/http/v1/dto"
	he "auction-platform/internal/controller/http/v1/errors"
	"auction-platform/internal/metrics"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	global  *rate.Limiter
	perIP   map[string]*rate.Limiter
	mu      sync.RWMutex
	rps     rate.Limit
	burst   int
	metrics *metrics.Metrics
}

func NewRateLimiter(rps float64, burst int, m *metrics.Metrics) *RateLimiter {
	return &RateLimiter{
		global:  rate.NewLimiter(rate.Limit(rps), burst),
		perIP:   make(map[string]*rate.Limiter),
		rps:     rate.Limit(rps / 10),
		burst:   burst / 10,
		metrics: m,
	}
}

func (rl *RateLimiter) getIPLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.perIP[ip]
	rl.mu.RUnlock()
	if exists {
		return limiter
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if limiter, exists = rl.perIP[ip]; exists {
		return limiter
	}
	limiter = rate.NewLimiter(rl.rps, rl.burst)
	rl.perIP[ip] = limiter
	return limiter
}

func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !rl.global.Allow() {
				rl.metrics.RateLimiterRejected.Inc()
				log.Warnf("Global rate limit: ip=%s path=%s", c.RealIP(), c.Path())
				return c.JSON(http.StatusTooManyRequests, hd.ErrorOutput{
					Error: hd.APIError{Code: he.ErrCodeRateLimited, Message: "too many requests"},
				})
			}

			if !rl.getIPLimiter(c.RealIP()).Allow() {
				rl.metrics.RateLimiterRejected.Inc()
				log.Warnf("Per-IP rate limit: ip=%s path=%s", c.RealIP(), c.Path())
				return c.JSON(http.StatusTooManyRequests, hd.ErrorOutput{
					Error: hd.APIError{Code: he.ErrCodeRateLimited, Message: "too many requests from your IP"},
				})
			}

			return next(c)
		}
	}
}
