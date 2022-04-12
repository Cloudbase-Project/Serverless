package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/Cloudbase-Project/serverless/dtos"
	"github.com/Cloudbase-Project/serverless/services"
	"github.com/Cloudbase-Project/serverless/utils"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

type ConfigHandler struct {
	l       *log.Logger
	service *services.ConfigService
}

// create new function
func NewConfigHandler(
	l *log.Logger,
	s *services.ConfigService,
) *ConfigHandler {
	return &ConfigHandler{l: l, service: s}
}

func (c *ConfigHandler) CreateConfig(rw http.ResponseWriter, r *http.Request) {
	var data *dtos.CreateConfigDTO
	utils.FromJSON(r.Body, &data)
	if _, err := dtos.Validate(data); err != nil {
		http.Error(rw, "Validation error : "+err.Error(), 400)
		return
	}

	config := c.service.CreateConfig(data)
	config.ToJSON(rw)
}

func (c *ConfigHandler) ToggleService(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	projectId := vars["projectId"]

	token := rw.Header().Get("authorization")
	if token == "" {
		http.Error(rw, "Token missing", 401)
	}

	tokenData, err := jwt.Parse(
		token,
		func(t *jwt.Token) (interface{}, error) { return os.Getenv("MAIN_SECRET_TOKEN"), nil },
	)
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	if !tokenData.Valid {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	claims, ok := tokenData.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(rw, "Error token", 400)
	}

	ownerId := claims["id"].(string)

	config, err := c.service.ToggleService(projectId, ownerId)
	config.ToJSON(rw)

}
