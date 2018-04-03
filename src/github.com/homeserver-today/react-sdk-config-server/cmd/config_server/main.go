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
)

const UnkErrJson = `{"code":"M_UNKNOWN","message":"Unexpected error processing response"}`

type requestCounter struct {
	lastId int
}

type Handler struct {
	h    func(http.ResponseWriter, *http.Request, *logrus.Entry) interface{}
	opts HandlerOpts
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

	logrus.Info("Starting media repository...")

	counter := requestCounter{}
	hOpts := HandlerOpts{&counter}

	optionsHandler := Handler{optionsRequest, hOpts}
	serveConfigHandler := Handler{serve.GetConfig, hOpts}
	setConfigHandler := Handler{rest.SetConfig, hOpts}
	deleteConfigHandler := Handler{rest.DeleteConfig, hOpts}

	routes := list.New()
	routes.PushBack(&ApiRoute{"/config.{domain:.*}.json", "GET", serveConfigHandler})
	routes.PushBack(&ApiRoute{"/config.json", "GET", serveConfigHandler})
	routes.PushBack(&ApiRoute{"/api/v1/config/{domain:.*}", "GET", serveConfigHandler})
	routes.PushBack(&ApiRoute{"/api/v1/config/{domain:.*}", "PUT", setConfigHandler})
	routes.PushBack(&ApiRoute{"/api/v1/config/{domain:.*}", "DELETE", deleteConfigHandler})

	for e := routes.Front(); e != nil; e = e.Next() {
		route := e.Value.(*ApiRoute)
		logrus.Info("Registering route: " + route.Method + " " + route.Path)
		rtr.Handle(route.Path, route.Handler).Methods(route.Method)
		rtr.Handle(route.Path, optionsHandler).Methods("OPTIONS")
	}

	rtr.NotFoundHandler = Handler{api.NotFoundHandler, hOpts}
	rtr.MethodNotAllowedHandler = Handler{api.MethodNotAllowedHandler, hOpts}

	address := config.Get().General.BindAddress + ":" + strconv.Itoa(config.Get().General.Port)
	http.Handle("/", rtr)

	logrus.WithField("address", address).Info("Started up. Listening at http://" + address)
	http.ListenAndServe(address, nil)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	b, err := json.Marshal(res)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, UnkErrJson, http.StatusInternalServerError)
		return
	}
	jsonStr := string(b)

	contextLog.Info("Replying with result: " + reflect.TypeOf(res).Elem().Name() + " " + jsonStr)

	switch result := res.(type) {
	case *api.ErrorResponse:
		w.Header().Set("Content-Type", "application/json")
		switch result.InternalCode {
		case "M_UNKNOWN_TOKEN":
			http.Error(w, jsonStr, http.StatusForbidden)
			break
		case "M_NOT_FOUND":
			http.Error(w, jsonStr, http.StatusNotFound)
			break
		case "M_BAD_REQUEST":
			http.Error(w, jsonStr, http.StatusBadRequest)
			break
		case "M_METHOD_NOT_ALLOWED":
			http.Error(w, jsonStr, http.StatusMethodNotAllowed)
			break
		default: // M_UNKNOWN
			http.Error(w, jsonStr, http.StatusInternalServerError)
			break
		}
		break
	default:
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonStr)
		break
	}
}

func (c *requestCounter) GetNextId() string {
	strId := strconv.Itoa(c.lastId)
	c.lastId = c.lastId + 1

	return "REQ-" + strId
}

func optionsRequest(w http.ResponseWriter, r *http.Request, log *logrus.Entry) interface{} {
	return &EmptyResponse{}
}
