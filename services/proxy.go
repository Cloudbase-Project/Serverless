package services

import (
	"log"

	"github.com/Cloudbase-Project/serverless/models"
	"gorm.io/gorm"
)

type ProxyService struct {
	db *gorm.DB
	l  *log.Logger
}

func NewProxyService(db *gorm.DB, l *log.Logger) *ProxyService {
	return &ProxyService{db: db, l: l}
}

func (ps *ProxyService) VerifyFunction(functionId string) (*models.Function, error) {
	var function models.Function
	if err := ps.db.Where(&models.Function{ID: function.ID}).First(&function).Error; err != nil {
		return nil, err
	}
	return &function, nil

}
