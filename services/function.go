package services

import (
	"errors"

	"github.com/Cloudbase-Project/serverless/constants"
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

func (fs *FunctionService) CreateFunction(
	code string,
	language constants.Language,
	userId string,
) (*models.Function, error) {

	var function models.Function
	if err := fs.db.Create(&models.Function{Code: code, Language: string(language), UserId: userId}).Error; err != nil {
		return nil, err
	}
	return &function, nil
}

type UpdateBuildStatusOptions struct {
	Function *models.Function
	Status   string
	Reason   *string
}

func (fs *FunctionService) UpdateBuildStatus(data UpdateBuildStatusOptions) {
	data.Function.BuildStatus = data.Status
	if data.Reason != nil {
		data.Function.BuildFailReason = *data.Reason
	}
	fs.db.Save(data.Function)
}

func (fs *FunctionService) SaveFunction(function *models.Function) {

	fs.db.Save(function)
}
