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
	GenderUUID    *uuid.UUID `json:"gender_uuid,omitempty"`
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
	UUID          uuid.UUID `db:"uuid" json:"uuid"`
	UserUUID      uuid.UUID `db:"user_uuid" json:"user_uuid"`
	Name          *string   `db:"name" json:"name,omitempty"`
	NIS           *string   `db:"nis" json:"nis,omitempty"`
	NISN          *string   `db:"nisn" json:"nisn,omitempty"`
	ClassName     *string   `db:"class_name" json:"class_name,omitempty"`
	Phone         *string   `db:"phone" json:"phone,omitempty"`
	Email         *string   `db:"email" json:"email,omitempty"`
	GenderName    *string   `db:"gender_name" json:"gender_name,omitempty"`
	Status        string    `db:"status" json:"status"`
	Address       *string   `db:"address" json:"address,omitempty"`
	ParentName    *string   `db:"parent_name" json:"parent_name,omitempty"`
	ParentEmail   *string   `db:"parent_email" json:"parent_email,omitempty"`
	ParentPhone   *string   `db:"parent_phone" json:"parent_phone,omitempty"`
	ParentAddress *string   `db:"parent_address" json:"parent_address,omitempty"`
}
