package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/homeserver-today/react-sdk-config-server/config"
	"github.com/sirupsen/logrus"
	"strconv"
)

var requestCount *prometheus.CounterVec
var responseCount *prometheus.CounterVec
var cacheEntries *prometheus.GaugeVec
var requestDuration *prometheus.SummaryVec
var cacheHitCount *prometheus.CounterVec
var cacheMissCount *prometheus.CounterVec

func initMetrics() {
	logrus.Info("Creating metrics...")

	requestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "config_server_requests_total",
		Help: "Number of requests",
	}, []string{"method", "handler"})
	prometheus.MustRegister(requestCount)

	responseCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "config_server_responses_total",
		Help: "Number of responses",
	}, []string{"method", "handler", "code"})
	prometheus.MustRegister(responseCount)

	cacheEntries = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "config_server_cache_entry_count",
		Help: "Number of items in the cache",
	}, []string{"name"})
	prometheus.MustRegister(cacheEntries)

	requestDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "config_server_request_duration_seconds",
		Help: "Time in seconds to process requests",
	}, []string{"method", "handler", "code"})
	prometheus.MustRegister(requestDuration)

	cacheHitCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "config_server_cache_hit_count",
		Help: "Number of hits on the cache",
	}, []string{"name"})
	prometheus.MustRegister(cacheHitCount)

	cacheMissCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "config_server_cache_miss_count",
		Help: "Number of misses on the cache",
	}, []string{"name"})
	prometheus.MustRegister(cacheMissCount)
}

func IncRequest(method string, handlerName string) {
	if !config.Get().Metrics.Enabled {
		return
	}

	requestCount.With(prometheus.Labels{
		"method":  method,
		"handler": handlerName,
	}).Inc()
}

func IncResponse(method string, handlerName string, code int) {
	if !config.Get().Metrics.Enabled {
		return
	}

	responseCount.With(prometheus.Labels{
		"method":  method,
		"handler": handlerName,
		"code":    strconv.Itoa(code),
	}).Inc()
}

func SetCacheCount(name string, size int) {
	if !config.Get().Metrics.Enabled {
		return
	}

	cacheEntries.With(prometheus.Labels{
		"name": name,
	}).Set(float64(size))
}

func IncCacheHit(name string) {
	if !config.Get().Metrics.Enabled {
		return
	}

	cacheHitCount.With(prometheus.Labels{
		"name": name,
	}).Inc()
}

func IncCacheMiss(name string) {
	if !config.Get().Metrics.Enabled {
		return
	}

	cacheMissCount.With(prometheus.Labels{
		"name": name,
	}).Inc()
}
