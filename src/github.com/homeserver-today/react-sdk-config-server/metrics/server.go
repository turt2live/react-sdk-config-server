package metrics

import (
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/homeserver-today/react-sdk-config-server/config"
	"strconv"
	"github.com/gorilla/mux"
)

func InitServer(rtr *mux.Router) {
	initMetrics()

	http.Handle("/api/v1/health/metrics", promhttp.Handler())
	rtr.Handle("/api/v1/health/metrics", promhttp.Handler())
	http.Handle("/api/v1/health/ping", PingHandler{})

	address := config.Get().Metrics.BindAddress + ":" + strconv.Itoa(config.Get().Metrics.Port)
	logrus.Info("Metrics listening on " + address)
	logrus.Fatal(http.ListenAndServe(address, nil))
}
