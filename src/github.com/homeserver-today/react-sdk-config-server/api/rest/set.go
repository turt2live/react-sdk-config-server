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
	"strings"
	"strconv"
)

func SetConfig(w http.ResponseWriter, r *http.Request, log *logrus.Entry) interface{} {
	if !isValidApiToken(r, log) {
		return api.AuthFailed()
	}

	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read body", err)
		return api.InternalServerError("Internal Server Error")
	}

	params := mux.Vars(r)

	keyPath := params["keyPath"]
	domain := params["domain"]

	if domain == "" {
		log.Warn("No domain in request")
		return api.BadRequest("No value given for 'domain'")
	}

	log = log.WithFields(logrus.Fields{
		"domain":  domain,
		"keyPath": keyPath,
	})

	bodyStr := string(bodyBytes)
	newConfig := models.ReactConfig{}

	if keyPath != "" {
		var targetVal interface{}

		// Try to infer the type, starting with numbers
		f, e := strconv.ParseFloat(bodyStr, 64)
		if e == nil {
			log.Info("Value interpreted as a number (float64)")
			targetVal = f
		} else {
			// Try booleans next
			if strings.ToLower(bodyStr) == "true" {
				log.Info("Value interpreted as boolean true")
				targetVal = true
			} else if strings.ToLower(bodyStr) == "false" {
				log.Info("Value interpreted as boolean false")
				targetVal = false
			} else {
				// Now we just try JSON or text
				contentType := r.Header.Get("Content-Type")
				if contentType != "application/json" {
					log.Info("Value interpreted as a string")
					targetVal = bodyStr
				} else {
					log.Info("Value interpreted as JSON")
					targetVal = models.ReactConfig{}
					err = json.Unmarshal(bodyBytes, &targetVal)
					if err != nil {
						log.Error("Failed to parse body as JSON", err)
						return api.BadRequest("Body not JSON")
					}
				}
			}
		}

		log.Info("Key path provided - performing lookup")

		conf, err := storage.GetForwardingCache(r.Context(), log).GetConfig(domain)
		if err != nil {
			log.Error("Error retrieving current configuration")
			return api.InternalServerError("Error retrieving")
		}

		// This is a fairly complicated loop that creates a new map for us to apply
		// on top of the current domain's config. This uses a similar approach to
		// traversing a linked list.
		parts := strings.Split(keyPath, "/")
		newValue := make(map[string]interface{})
		lastObject := &newValue
		var lastObjectParent *map[string]interface{}
		for _, k := range parts {
			log.Info("Building path: ", k)
			newVal := make(map[string]interface{})
			(*lastObject)[k] = newVal
			lastObjectParent = lastObject
			lastObject = &newVal
		}

		// This is why we kept track of the lastObject's parent: so we can set the new
		// real value in the tree. This does mean that the last lastObject generated will
		// be waste, but we are okay to throw that to the garbage collector.
		(*lastObjectParent)[parts[len(parts)-1]] = targetVal

		// Overwrite the values in the config and assign newConfig to our combined config.
		conf.TakeFrom(newValue)
		newConfig = *conf
	} else {
		// Setting the whole config requires the content to be JSON
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			return api.BadRequest("Body not JSON")
		}

		err = json.Unmarshal(bodyBytes, &newConfig)
		if err != nil {
			log.Error("Failed to parse body as JSON", err)
			return api.BadRequest("Body not JSON")
		}
	}

	newConf, err := storage.GetForwardingCache(r.Context(), log).SetConfig(domain, &newConfig)
	if err != nil {
		log.Error("Error updating config", err)
		return api.InternalServerError("Error updating config")
	}

	return newConf
}
