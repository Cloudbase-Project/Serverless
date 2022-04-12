package services

import (
	"fmt"
	"log"

	"github.com/Cloudbase-Project/serverless/dtos"
	"github.com/Cloudbase-Project/serverless/models"
	"gorm.io/gorm"
)

type ConfigService struct {
	db *gorm.DB
	l  *log.Logger
}

func NewConfigService(db *gorm.DB, l *log.Logger) *ConfigService {
	return &ConfigService{db: db, l: l}
}

func (cs *ConfigService) CreateConfig(
	CreateConfigDTO *dtos.CreateConfigDTO,
) *models.Config {
	config := models.Config{Owner: CreateConfigDTO.Owner, ProjectId: CreateConfigDTO.ProjectId}
	result := cs.db.Create(&config)
	fmt.Printf("config created: %v\n", &result)
	return &config
}

func (cs *ConfigService) ToggleService(projectId string, ownerId string) (*models.Config, error) {
	var config models.Config

	if err := cs.db.Where(&models.Config{Owner: ownerId, ProjectId: projectId}, &config).Error; err != nil {
		return nil, err
	}
	config.Enabled = !config.Enabled
	cs.db.Save(&config)
	return &config, nil
}
