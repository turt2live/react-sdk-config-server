package rest

import (
	"net/http"
	"github.com/sirupsen/logrus"
	"github.com/homeserver-today/react-sdk-config-server/api"
	"github.com/gorilla/mux"
	"github.com/homeserver-today/react-sdk-config-server/storage"
)

func DeleteConfig(w http.ResponseWriter, r *http.Request, log *logrus.Entry) interface{} {
	if !isValidApiToken(r, log) {
		return api.AuthFailed()
	}

	params := mux.Vars(r)

	keyPath := params["keyPath"]
	domain := params["domain"]

	if domain == "" {
		log.Warn("No domain in request")
		return api.BadRequest("No value given for 'domain'")
	}

	log = log.WithFields(logrus.Fields{
		"domain": domain,
	})

	if keyPath != "" {
		log.Info("Key path provided - performing lookup")

		return api.InternalServerError("Not yet implemented")
	} else {
		newConf, err := storage.GetForwardingCache(r.Context(), log).DeleteConfig(domain)
		if err != nil {
			log.Error("Error deleting config", err)
			return api.InternalServerError("Error deleting config")
		}

		return newConf
	}
}
