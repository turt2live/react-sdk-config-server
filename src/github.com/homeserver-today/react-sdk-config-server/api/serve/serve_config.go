package serve

import (
	"net/http"
	"github.com/sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/homeserver-today/react-sdk-config-server/storage"
	"database/sql"
	"github.com/homeserver-today/react-sdk-config-server/api"
)

func GetConfig(w http.ResponseWriter, r *http.Request, log *logrus.Entry) interface{} {
	params := mux.Vars(r)

	domain := params["domain"]
	if domain == "" {
		log.Warn("No domain specified in request - assuming Host as the default")
		domain = r.Host
	}

	log = log.WithFields(logrus.Fields{
		"domain": domain,
	})

	// First try to get the requested domain's config
	conf, err := storage.GetDatabase().GetConfig(r.Context(), domain)
	if err == sql.ErrNoRows {
		log.Warn("Failed to find config (ErrNoRows) - trying default")
		conf, err = storage.GetDatabase().GetConfig(r.Context(), "default")
	}

	// We may have requested the default config, so check the error again
	if err == sql.ErrNoRows {
		log.Warn("Failed to find default config (ErrNoRows)")
		return api.NotFoundError()
	} else if err != nil {
		log.Error("Error looking up config", err)
		return api.InternalServerError("Unknown error")
	}

	// No errors, so return the config as-is
	return conf
}
