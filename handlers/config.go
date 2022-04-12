package handlers

import (
	"log"
	"net/http"

	"github.com/Cloudbase-Project/serverless/dtos"
	"github.com/Cloudbase-Project/serverless/services"
	"github.com/Cloudbase-Project/serverless/utils"
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
