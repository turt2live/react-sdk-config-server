package rest

import (
	"net/http"
	"github.com/homeserver-today/react-sdk-config-server/config"
	"github.com/sirupsen/logrus"
	"strings"
)

func isValidApiToken(r *http.Request, log *logrus.Entry) (bool) {
	requiredToken := config.Get().ApiConfig.SharedSecret
	if requiredToken == config.DefaultSharedSecret {
		log.Warn("Your API token is set to the default value. Please change this to enable requests.")
		return false
	}

	sentToken := r.Header.Get("Authorization")
	if !strings.HasPrefix(sentToken, "Bearer ") {
		log.Warn("Authorization header is not a Bearer Token")
		return false
	}

	sentToken = sentToken[len("Bearer "):]
	if sentToken != requiredToken {
		log.Warn("Token does not match configuration")
		return false
	}

	log.Info("Authorized request (tokens match)")
	return true
}
