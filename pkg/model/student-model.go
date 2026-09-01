package model

import "github.com/google/uuid"

type CreateStudentModel struct {
	Name          *string    `json:"name,omitempty"`
	NIS           *string    `json:"nis,omitempty"`
	NISN          *string    `json:"nisn,omitempty"`
	Email         *string    `json:"email,omitempty"`
	Phone         *string    `json:"phone,omitempty"`
	Address       *string    `json:"address,omitempty"`
	ClassUUID     *uuid.UUID `json:"class_uuid,omitempty"`
	ImgLocation   *string    `json:"img_location,omitempty"`
	ParentName    *string    `json:"parent_name,omitempty"`
	ParentEmail   *string    `json:"parent_email,omitempty"`
	ParentPhone   *string    `json:"parent_phone,omitempty"`
	ParentAddress *string    `json:"parent_address,omitempty"`
}

type UpdateStudentModel struct {
	UUID        uuid.UUID `json:"uuid"`
	Name        *string   `json:"name,omitempty"`
	Email       *string   `json:"email,omitempty"`
	Phone       *string   `json:"phone,omitempty"`
	Address     *string   `json:"address,omitempty"`
	ImgLocation *string   `json:"img_location,omitempty"`
	RoleUUID    uuid.UUID `json:"role_uuid"`
	Version     *string   `json:"version,omitempty"`
	Status      string    `json:"status,omitempty"`
}

type DeleteStudentModel struct {
	UUID uuid.UUID `json:"uuid"`
}

type ReadStudentModelResult struct {
	UUID        uuid.UUID `db:"uuid" json:"uuid"`
	Name        *string   `db:"name" json:"name,omitempty"`
	Email       *string   `db:"email" json:"email,omitempty"`
	Phone       *string   `db:"phone" json:"phone,omitempty"`
	Address     *string   `db:"address" json:"address,omitempty"`
	ImgLocation *string   `db:"img_location" json:"img_location,omitempty"`
	RoleUUID    uuid.UUID `db:"role_uuid" json:"role_uuid"`
	RoleName    string    `db:"role_name" json:"role_name"`
	Status      string    `db:"status" json:"status"`
	Version     *string   `db:"version" json:"version,omitempty"`
}
