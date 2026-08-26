package model

import (
	"time"

	"github.com/google/uuid"
)

type StatusModel struct {
	UUID        uuid.UUID  `db:"uuid"`
	Name        string     `db:"name"`
	AbbrName    string     `db:"abbr_name"`
	CreatedDate time.Time  `db:"created_date"`
	UpdatedDate *time.Time `db:"updated_date"`
}
