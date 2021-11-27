package dtos

import (
	"github.com/Cloudbase-Project/serverless/constants"
	"github.com/asaskevich/govalidator"
)

func Validate(s interface{}) (bool, error) {
	// validate := validator.New()

	govalidator.SetFieldsRequiredByDefault(true)

	return govalidator.ValidateStruct(s)

}

type PostCodeDTO struct {
	Code     string             `valid:"required;type(string)"`
	Language constants.Language `valid:"required;type(string)"`
}
