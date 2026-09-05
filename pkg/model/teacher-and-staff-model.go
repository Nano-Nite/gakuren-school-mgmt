package model

import (
	"time"

	"github.com/google/uuid"
)

// type StudentModel struct {
// 	UUID          uuid.UUID  `db:"uuid" json:"uuid"`
// 	UserUUID      uuid.UUID  `db:"user_uuid" json:"user_uuid"`
// 	Name          *string    `db:"name" json:"name,omitempty"`
// 	NIS           *string    `db:"nis" json:"nis,omitempty"`
// 	NISN          *string    `db:"nisn" json:"nisn,omitempty"`
// 	Email         *string    `db:"email" json:"email,omitempty"`
// 	Phone         *string    `db:"phone" json:"phone,omitempty"`
// 	Address       *string    `db:"address" json:"address,omitempty"`
// 	ClassUUID     *uuid.UUID `db:"class_uuid" json:"class_uuid,omitempty"`
// 	GenderUUID    *uuid.UUID `db:"gender_uuid" json:"gender_uuid,omitempty"`
// 	ImgLocation   *string    `db:"img_location" json:"img_location,omitempty"`
// 	ParentName    *string    `db:"parent_name" json:"parent_name,omitempty"`
// 	ParentEmail   *string    `db:"parent_email" json:"parent_email,omitempty"`
// 	ParentPhone   *string    `db:"parent_phone" json:"parent_phone,omitempty"`
// 	ParentAddress *string    `db:"parent_address" json:"parent_address,omitempty"`
// 	StatusUUID    uuid.UUID  `db:"status_uuid" json:"status_uuid,omitempty"`
// }

type CreateTNSModel struct {
	Biodata        TNSBioModel         `db:"biodata" json:"biodata,omitempty"`
	EducationLevel []TNSEducationModel `db:"education_level" json:"education_level,omitempty"`
	IsStaff        bool                `db:"is_staff" json:"is_staff,omitempty"`
	NIK            string              `db:"nik" json:"nik,omitempty"`
	NUPTK          *string             `db:"nuptk" json:"nuptk,omitempty"`
	NIP            *string             `db:"nip" json:"nip,omitempty"`
	Postions       *uuid.UUIDs         `db:"postions" json:"postions,omitempty"`
	Titles         *uuid.UUIDs         `db:"titles" json:"titles,omitempty"`
	Subjects       *uuid.UUIDs         `db:"subjects" json:"subjects,omitempty"`
	JoinDate       *time.Time          `db:"joindate" json:"joindate,omitempty"`
	ResignDate     *time.Time          `db:"resign_date" json:"resign_date,omitempty"`
	StatusUUID     uuid.UUID           `db:"status_uuid" json:"status_uuid,omitempty"`
	CreatedDate    time.Time           `db:"created_date" json:"created_date,omitempty"`
	UpdatedDate    *time.Time          `db:"updated_date" json:"updated_date,omitempty"`
}

type TNSBioModel struct {
	Fullname   string       `db:"full_name" json:"full_name,omitempty"`
	Title      *[]uuid.UUID `db:"title" json:"title,omitempty"`
	Email      string       `db:"email" json:"email,omitempty"`
	Phone      string       `db:"phone" json:"phone,omitempty"`
	GenderUUID uuid.UUID    `db:"gender_uuid" json:"gender_uuid,omitempty"`
	BirthPlace string       `db:"birth_place" json:"birth_place,omitempty"`
	BirthDate  time.Time    `db:"birth_date" json:"birth_date,omitempty"`
	Address    string       `db:"address" json:"address,omitempty"`
}

type TNSEducationModel struct {
	LastEducation      bool      `db:"last_education" json:"last_education,omitempty"`
	InstitutionName    string    `db:"institution_name" json:"institution_name,omitempty"`
	EducationLevelUUID uuid.UUID `db:"education_level_uuid" json:"education_level_uuid,omitempty"`
	Major              string    `db:"major" json:"major,omitempty"`
	StartYear          time.Time `db:"start_year" json:"start_year,omitempty"`
	EndYear            time.Time `db:"end_year" json:"end_year,omitempty"`
}

// type UpdateStudentModel struct {
// 	UUID          uuid.UUID  `json:"uuid"`
// 	Name          *string    `json:"name,omitempty"`
// 	NIS           *string    `json:"nis,omitempty"`
// 	NISN          *string    `json:"nisn,omitempty"`
// 	Email         *string    `json:"email,omitempty"`
// 	Phone         *string    `json:"phone,omitempty"`
// 	Address       *string    `json:"address,omitempty"`
// 	ClassUUID     *uuid.UUID `json:"class_uuid,omitempty"`
// 	GenderUUID    *uuid.UUID `json:"gender_uuid,omitempty"`
// 	ImgLocation   *string    `json:"img_location,omitempty"`
// 	ParentName    *string    `json:"parent_name,omitempty"`
// 	ParentEmail   *string    `json:"parent_email,omitempty"`
// 	ParentPhone   *string    `json:"parent_phone,omitempty"`
// 	ParentAddress *string    `json:"parent_address,omitempty"`
// 	Status        uuid.UUID  `json:"status,omitempty"`
// }

// type DeleteStudentModel struct {
// 	UUID uuid.UUID `json:"uuid"`
// }

// type ReadStudentModelResult struct {
// 	UUID          uuid.UUID `db:"uuid" json:"uuid"`
// 	UserUUID      uuid.UUID `db:"user_uuid" json:"user_uuid"`
// 	Name          *string   `db:"name" json:"name,omitempty"`
// 	NIS           *string   `db:"nis" json:"nis,omitempty"`
// 	NISN          *string   `db:"nisn" json:"nisn,omitempty"`
// 	ClassName     *string   `db:"class_name" json:"class_name,omitempty"`
// 	Phone         *string   `db:"phone" json:"phone,omitempty"`
// 	Email         *string   `db:"email" json:"email,omitempty"`
// 	GenderName    *string   `db:"gender_name" json:"gender_name,omitempty"`
// 	Status        string    `db:"status" json:"status"`
// 	Address       *string   `db:"address" json:"address,omitempty"`
// 	ParentName    *string   `db:"parent_name" json:"parent_name,omitempty"`
// 	ParentEmail   *string   `db:"parent_email" json:"parent_email,omitempty"`
// 	ParentPhone   *string   `db:"parent_phone" json:"parent_phone,omitempty"`
// 	ParentAddress *string   `db:"parent_address" json:"parent_address,omitempty"`
// }
