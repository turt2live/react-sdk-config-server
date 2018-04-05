package metrics

import (
	"time"
	"github.com/homeserver-today/react-sdk-config-server/config"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
)

type requestTracker struct {
	start   time.Time
	method  string
	handler string
}

func StartRequestTimer(method string, handler string) (*requestTracker) {
	return &requestTracker{
		start:   time.Now(),
		method:  method,
		handler: handler,
	}
}

func (r requestTracker) End(code int) (time.Duration) {
	duration := time.Since(r.start)

	if !config.Get().Metrics.Enabled {
		return duration
	}

	requestDuration.With(prometheus.Labels{
		"method":  r.method,
		"handler": r.handler,
		"code":    strconv.Itoa(code),
	}).Observe(duration.Seconds())
	return duration
}
