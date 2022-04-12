package dtos

import (
	"github.com/Cloudbase-Project/serverless/constants"
)

type PostCodeDTO struct {
	Code     string             `valid:"required;type(string)"`
	Language constants.Language `valid:"required;type(string)"`
}

type UpdateCodeDTO struct {
	Code string `valid:"required;type(string)"`
}
