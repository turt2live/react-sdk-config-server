package rest

import (
	"net/http"
	"github.com/sirupsen/logrus"
	"github.com/homeserver-today/react-sdk-config-server/api"
	"github.com/gorilla/mux"
	"github.com/homeserver-today/react-sdk-config-server/storage"
	"github.com/homeserver-today/react-sdk-config-server/models"
)

func DeleteConfig(w http.ResponseWriter, r *http.Request, log *logrus.Entry) interface{} {
	if !isValidApiToken(r, log) {
		return api.AuthFailed()
	}

	params := mux.Vars(r)

	domain := params["domain"]
	if domain == "" {
		log.Warn("No domain in request")
		return api.BadRequest("No value given for 'domain'")
	}

	log = log.WithFields(logrus.Fields{
		"domain": domain,
	})

	err := storage.GetDatabase().DeleteConfig(r.Context(), domain)
	if err != nil {
		log.Error("Error deleting config", err)
		return api.InternalServerError("Error deleting config")
	}

	return &models.ReactConfig{}
}
