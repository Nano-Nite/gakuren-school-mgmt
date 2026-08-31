package model

import (
	"time"

	"github.com/google/uuid"
)

type ClassModel struct {
	UUID            *uuid.UUID `db:"uuid" json:"uuid"`
	Name            string     `db:"name" json:"name"`
	AbbrName        *string    `db:"abbr_name" json:"abbr_name"`
	Level           int        `db:"level" json:"level"`
	HomeroomTeacher *uuid.UUID `db:"homeroom_teacher" json:"homeroom_teacher"`
	StatusUUID      uuid.UUID  `db:"status_uuid" json:"status_uuid"`
	CreatedDate     time.Time  `db:"created_date" json:"created_date"`
	UpdatedDate     *time.Time `db:"updated_date" json:"updated_date"`
	TenantUUID      uuid.UUID  `db:"tenant_uuid" json:"tenant_uuid"`
}

type CreateClassModel struct {
	Name            string     `json:"name"`
	AbbrName        *string    `json:"abbr_name,omitempty"`
	Level           int        `json:"level"`
	HomeroomTeacher *uuid.UUID `json:"homeroom_teacher,omitempty"`
	TenantUUID      *uuid.UUID `json:"tenant_uuid"`
}

type UpdateClassModel struct {
	UUID            uuid.UUID  `json:"uuid"`
	Name            string     `json:"name"`
	AbbrName        *string    `json:"abbr_name,omitempty"`
	Level           int        `json:"level"`
	HomeroomTeacher *uuid.UUID `json:"homeroom_teacher,omitempty"`
	Status          string     `json:"status,omitempty"`
}

type DeleteClassModel struct {
	UUID uuid.UUID `json:"uuid"`
}

type ReadClassModelResult struct {
	UUID            uuid.UUID `db:"uuid" json:"uuid"`
	Name            string    `db:"name" json:"name"`
	AbbrName        *string   `db:"abbr_name" json:"abbr_name"`
	Level           int       `db:"level" json:"level"`
	HomeroomTeacher *string   `db:"homeroom_teacher" json:"homeroom_teacher"`
	Status          string    `db:"status" json:"status"`
	TotalStudent    int       `db:"total_student" json:"total_student"`
}
