package rest

import (
	"net/http"
	"github.com/sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/homeserver-today/react-sdk-config-server/storage"
	"github.com/homeserver-today/react-sdk-config-server/api"
	"strings"
	"github.com/homeserver-today/react-sdk-config-server/models"
)

func GetConfig(w http.ResponseWriter, r *http.Request, log *logrus.Entry) interface{} {
	if !isValidApiToken(r, log) {
		return api.AuthFailed()
	}

	params := mux.Vars(r)

	keyPath := params["keyPath"]
	domain := params["domain"]

	if domain == "" {
		log.Warn("No domain specified in request - assuming Host as the default")
		domain = r.Host
	}

	log = log.WithFields(logrus.Fields{
		"domain":  domain,
		"keyPath": keyPath,
	})

	log.Info("Getting config")
	conf, err := storage.GetForwardingCache(r.Context(), log).GetConfig(domain)
	if err != nil {
		log.Error("Error retrieving configuration", err)
		return api.InternalServerError("Error retrieving config")
	}

	if keyPath != "" {
		log.Info("Key path provided - performing lookup")
		parts := strings.Split(keyPath, "/")
		var currentMap interface{} = *conf
		for _, k := range parts {
			log.Info("Finding path: ", k)

			var target interface{}
			var m map[string]interface{}
			var ok bool
			if m, ok = currentMap.(map[string]interface{}); ok {
				target = m[k]
			} else if m, ok = currentMap.(models.ReactConfig); ok {
				target = m[k]
			} else {
				return api.BadRequest("Path is not a map")
			}

			if target == nil {
				return api.NotFoundError()
			}
			currentMap = target
		}

		return &api.UnknownContentTypeResponse{Value: currentMap}
	}

	// No errors, so return the config as-is
	return conf
}
