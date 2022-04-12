package dtos

import "github.com/asaskevich/govalidator"

func Validate(s interface{}) (bool, error) {
	// validate := validator.New()

	govalidator.SetFieldsRequiredByDefault(true)

	return govalidator.ValidateStruct(s)

}
