package services

import (
	"errors"

	"github.com/Cloudbase-Project/serverless/models"
	"gorm.io/gorm"
)

type FunctionService struct {
	db *gorm.DB
}

func NewFunctionService(db *gorm.DB) *FunctionService {
	return &FunctionService{db: db}
}

func (fs *FunctionService) GetAllFunctions() (*models.Functions, error) {

	var functions models.Functions

	if err := fs.db.Where("userId = ?").Find(&functions).Error; err != nil {
		return nil, err
	}

	return &functions, nil
}

func (fs *FunctionService) GetFunction(codeId string) (*models.Function, error) {
	var function models.Function
	if err := fs.db.First(&function, "id = ?", codeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &function, nil
}
