package rest

import (
	"net/http"
	"github.com/sirupsen/logrus"
	"github.com/homeserver-today/react-sdk-config-server/api"
	"encoding/json"
	"io/ioutil"
	"github.com/gorilla/mux"
	"github.com/homeserver-today/react-sdk-config-server/models"
	"github.com/homeserver-today/react-sdk-config-server/storage"
)

func SetConfig(w http.ResponseWriter, r *http.Request, log *logrus.Entry) interface{} {
	if !isValidApiToken(r, log) {
		return api.AuthFailed()
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return api.BadRequest("Body not JSON")
	}

	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read body", err)
		return api.InternalServerError("Internal Server Error")
	}

	newConfig := models.ReactConfig{}
	err = json.Unmarshal(bodyBytes, &newConfig)
	if err != nil {
		log.Error("Failed to parse body as JSON", err)
		return api.BadRequest("Body not JSON")
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

	err = storage.GetDatabase().UpsertConfig(r.Context(), domain, newConfig)
	if err != nil {
		log.Error("Error saving config", err)
		return api.InternalServerError("Error saving config")
	}

	return newConfig
}
