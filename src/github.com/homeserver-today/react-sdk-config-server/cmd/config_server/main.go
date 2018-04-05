package main

import (
	"net/http"
	"flag"
	"encoding/json"
	"strconv"
	"strings"
	"net"
	"reflect"
	"io"
	"github.com/sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/sebest/xff"
	"github.com/homeserver-today/react-sdk-config-server/api"
	"github.com/homeserver-today/react-sdk-config-server/config"
	"github.com/homeserver-today/react-sdk-config-server/logging"
	"github.com/homeserver-today/react-sdk-config-server/api/serve"
	"github.com/homeserver-today/react-sdk-config-server/api/rest"
	"container/list"
	"github.com/homeserver-today/react-sdk-config-server/models"
	"fmt"
	"github.com/homeserver-today/react-sdk-config-server/metrics"
)

const UnkErrJson = `{"code":"M_UNKNOWN","message":"Unexpected error processing response"}`

type requestCounter struct {
	lastId int
}

type Handler struct {
	h    func(http.ResponseWriter, *http.Request, *logrus.Entry) interface{}
	opts HandlerOpts
	name string
}

type HandlerOpts struct {
	reqCounter *requestCounter
}

type ApiRoute struct {
	Path    string
	Method  string
	Handler Handler
}

type EmptyResponse struct{}

func main() {
	configPath := flag.String("config", "config-server.yaml", "The path to the configuration")
	migrationsPath := flag.String("migrations", "./migrations", "The absolute path the migrations folder")
	flag.Parse()

	config.Path = *configPath
	config.Runtime.MigrationsPath = *migrationsPath

	rtr := mux.NewRouter()

	err := logging.Setup(config.Get().General.LogDirectory)
	if err != nil {
		panic(err)
	}

	logrus.Info("Starting config server...")

	counter := requestCounter{}
	hOpts := HandlerOpts{&counter}

	optionsHandler := Handler{optionsRequest, hOpts, "Options"}
	serveConfigHandler := Handler{serve.GetConfig, hOpts, "ServeConfig"}
	getConfigHandler := Handler{rest.GetConfig, hOpts, "GetConfig"}
	setConfigHandler := Handler{rest.SetConfig, hOpts, "SetConfig"}
	deleteConfigHandler := Handler{rest.DeleteConfig, hOpts, "DeleteConfig"}

	routes := list.New()
	routes.PushBack(&ApiRoute{"/config.{domain:.*}.json", "GET", serveConfigHandler})
	routes.PushBack(&ApiRoute{"/config.json", "GET", serveConfigHandler})
	routes.PushBack(&ApiRoute{"/api/v1/config/{domain:.*}", "GET", getConfigHandler})
	routes.PushBack(&ApiRoute{"/api/v1/config/{domain:.*}", "PUT", setConfigHandler})
	routes.PushBack(&ApiRoute{"/api/v1/config/{domain:.*}", "DELETE", deleteConfigHandler})

	for e := routes.Front(); e != nil; e = e.Next() {
		route := e.Value.(*ApiRoute)
		logrus.Info("Registering route: " + route.Method + " " + route.Path)
		rtr.Handle(route.Path, route.Handler).Methods(route.Method)
		rtr.Handle(route.Path, optionsHandler).Methods("OPTIONS")
	}

	logrus.Info("Registering route: GET /api/v1/health/ping")
	rtr.Handle("/api/v1/health/ping", &metrics.PingHandler{}).Methods("GET", "OPTIONS")

	rtr.NotFoundHandler = Handler{api.NotFoundHandler, hOpts, "NotFound"}
	rtr.MethodNotAllowedHandler = Handler{api.MethodNotAllowedHandler, hOpts, "MethodNotAllowed"}

	if config.Get().Metrics.Enabled {
		logrus.Info("Enabling metrics reporting (Prometheus)")
		go metrics.InitServer(rtr)

		// TODO: DISABLE METRICS (DO NOT PASS ROUTER)
		logrus.Warn("ENABLING METRICS ON MAIN HTTP SERVER FOR DEBUGGING PURPOSES")
	}

	address := config.Get().General.BindAddress + ":" + strconv.Itoa(config.Get().General.Port)
	http.Handle("/", rtr)

	logrus.Info("Started up. Listening at http://" + address)
	logrus.Fatal(http.ListenAndServe(address, nil))
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	timer := metrics.StartRequestTimer(r.Method, h.name)
	metrics.IncRequest(r.Method, h.name)
	requestComplete := func(statusCode int, log *logrus.Entry) {
		metrics.IncResponse(r.Method, h.name, statusCode)
		timeToComplete := timer.End(statusCode)
		log.Info("Request completed in ", timeToComplete, " (", timeToComplete.Seconds(), " seconds)")
	}

	isUsingForwardedHost := false
	if r.Header.Get("X-Forwarded-Host") != "" {
		r.Host = r.Header.Get("X-Forwarded-Host")
		isUsingForwardedHost = true
	}
	r.Host = strings.Split(r.Host, ":")[0]

	raddr := xff.GetRemoteAddr(r)
	host, _, err := net.SplitHostPort(raddr)
	if err != nil {
		logrus.Error(err)
		host = raddr
	}
	r.RemoteAddr = host

	contextLog := logrus.WithFields(logrus.Fields{
		"method":             r.Method,
		"host":               r.Host,
		"usingForwardedHost": isUsingForwardedHost,
		"resource":           r.URL.Path,
		"requestId":          h.opts.reqCounter.GetNextId(),
		"remoteAddr":         r.RemoteAddr,
	})
	contextLog.Info("Received request")

	// Send CORS and other basic headers
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public,max-age=3600,s-maxage=3600")
	w.Header().Set("Server", "react-sdk-config-server")

	// Process response
	res := h.h(w, r, contextLog)
	if res == nil {
		res = &EmptyResponse{}
	}

	if unk, ok := res.(*api.UnknownContentTypeResponse); ok {
		m, isMap := unk.Value.(map[string]interface{})
		c, isConf := unk.Value.(models.ReactConfig)
		a, isArray := unk.Value.([]interface{})

		if !isMap && !isConf && !isArray {
			asStr := fmt.Sprintf("%v", unk.Value)
			contextLog.Warn("Non-JSON response encountered: ", asStr)
			w.Header().Set("Content-Type", "text/plain")
			io.WriteString(w, asStr)
			requestComplete(http.StatusOK, contextLog)
			return
		} else if isMap {
			res = m
		} else if isConf {
			res = c
		} else if isArray {
			res = a
		}
	}

	b, err := json.Marshal(res)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, UnkErrJson, http.StatusInternalServerError)
		requestComplete(http.StatusOK, contextLog)
		return
	}
	jsonStr := string(b)

	contextLog.Info("Replying with result: " + reflect.TypeOf(res).Elem().Name() + " " + jsonStr)

	statusCode := http.StatusOK
	switch result := res.(type) {
	case *api.ErrorResponse:
		switch result.InternalCode {
		case "M_UNKNOWN_TOKEN":
			statusCode = http.StatusForbidden
			break
		case "M_NOT_FOUND":
			statusCode = http.StatusNotFound
			break
		case "M_BAD_REQUEST":
			statusCode = http.StatusBadRequest
			break
		case "M_METHOD_NOT_ALLOWED":
			statusCode = http.StatusMethodNotAllowed
			break
		default: // M_UNKNOWN
			statusCode = http.StatusInternalServerError
			break
		}
		break
	default:
		statusCode = http.StatusOK
		break
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	io.WriteString(w, jsonStr)
	requestComplete(statusCode, contextLog)
}

func (c *requestCounter) GetNextId() string {
	strId := strconv.Itoa(c.lastId)
	c.lastId = c.lastId + 1

	return "REQ-" + strId
}

func optionsRequest(w http.ResponseWriter, r *http.Request, log *logrus.Entry) interface{} {
	return &EmptyResponse{}
}
