package dtos

type CreateConfigDTO struct {
	Owner     string `valid:"required;type(string)"`
	ProjectId string `valid:"required;type(string)"`
}
