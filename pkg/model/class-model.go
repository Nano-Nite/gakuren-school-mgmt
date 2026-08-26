package model

import (
	"time"

	"github.com/google/uuid"
)

type ClassModel struct {
	UUID            uuid.UUID  `db:"uuid"`
	Name            string     `db:"name"`
	AbbrName        *string    `db:"abbr_name"`
	Level           int        `db:"level"`
	HomeroomTeacher *uuid.UUID `db:"homeroom_teacher"`
	StatusUUID      uuid.UUID  `db:"status_uuid"`
	CreatedDate     time.Time  `db:"created_date"`
	UpdatedDate     *time.Time `db:"updated_date"`
	TenantUUID      uuid.UUID  `db:"tenant_uuid"`
}

type CreateClassModel struct {
	Name            string     `json:"name"`
	AbbrName        *string    `json:"abbr_name,omitempty"`
	Level           int        `json:"level"`
	HomeroomTeacher *uuid.UUID `json:"homeroom_teacher,omitempty"`
	TenantUUID      *uuid.UUID `json:"tenant_uuid"`
}

type ReadClassModelResult struct {
	UUID            uuid.UUID `db:"uuid"`
	Name            string    `db:"name"`
	AbbrName        *string   `db:"abbr_name"`
	Level           int       `db:"level"`
	HomeroomTeacher *string   `db:"homeroom_teacher"`
	Status          string    `db:"status"`
	TotalStudent    int       `db:"total_student"`
}
