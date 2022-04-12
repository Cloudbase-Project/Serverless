package models

import (
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Config struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	CreatedAt time.Time      `                                                       json:"-"`         // auto populated by gorm
	UpdatedAt time.Time      `                                                       json:"-"`         // auto populated by gorm
	DeletedAt gorm.DeletedAt `gorm:"index"                                           json:"-"`         // auto populated by gorm
	ProjectId string         `                                                       json:"projectId"` // user table is controlled by cloudbase-main
	Owner     string         `                                                       json:"owner"`
	Enabled   bool           `                                                       json:"enabled"`
}

func (f *Config) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(f)
}
