package rest

import (
	"net/http"
	"github.com/sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/homeserver-today/react-sdk-config-server/storage"
	"github.com/homeserver-today/react-sdk-config-server/api"
)

func GetConfig(w http.ResponseWriter, r *http.Request, log *logrus.Entry) interface{} {
	if !isValidApiToken(r, log) {
		return api.AuthFailed()
	}

	params := mux.Vars(r)

	domain := params["domain"]
	if domain == "" {
		log.Warn("No domain specified in request - assuming Host as the default")
		domain = r.Host
	}

	log = log.WithFields(logrus.Fields{
		"domain": domain,
	})

	conf, err := storage.GetForwardingCache(r.Context(), log).GetConfig(domain)
	if err != nil {
		log.Error("Error retrieving configuration", err)
		return api.InternalServerError("Error retrieving config")
	}

	// No errors, so return the config as-is
	return conf
}
