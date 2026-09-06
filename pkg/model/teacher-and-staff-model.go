package model

import (
	"encoding/json"
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
	Biodata        TNSBioModel           `db:"biodata" json:"biodata,omitempty"`
	EducationLevel []TNSEducationModel   `db:"education_level" json:"education_level,omitempty"`
	ImgLocation    string                `db:"img_location" json:"img_location,omitempty"`
	IsStaff        bool                  `db:"is_staff" json:"is_staff,omitempty"`
	NIK            string                `db:"nik" json:"nik,omitempty"`
	NUPTK          *string               `db:"nuptk" json:"nuptk,omitempty"`
	NIP            *string               `db:"nip" json:"nip,omitempty"`
	Positions      []ResultPositionModel `db:"positions" json:"positions,omitempty"`
	Titles         []ResultTitleModel    `db:"titles" json:"titles,omitempty"`
	Subjects       []ResultSubjectModel  `db:"subjects" json:"subjects,omitempty"`
	JoinDate       *time.Time            `db:"join_date" json:"join_date,omitempty"`
	ResignDate     *time.Time            `db:"resign_date" json:"resign_date,omitempty"`
	StatusUUID     uuid.UUID             `db:"status_uuid" json:"status_uuid,omitempty"`
	CreatedDate    time.Time             `db:"created_date" json:"created_date,omitempty"`
	UpdatedDate    *time.Time            `db:"updated_date" json:"updated_date,omitempty"`
}

type TNSBioModel struct {
	Fullname   string    `db:"full_name" json:"full_name,omitempty"`
	Email      string    `db:"email" json:"email,omitempty"`
	Phone      string    `db:"phone" json:"phone,omitempty"`
	GenderUUID uuid.UUID `db:"gender_uuid" json:"gender_uuid,omitempty"`
	BirthPlace string    `db:"birth_place" json:"birth_place,omitempty"`
	BirthDate  time.Time `db:"birth_date" json:"birth_date,omitempty" time_format:"2006-01-02"`
	Address    string    `db:"address" json:"address,omitempty"`
}

type TNSEducationModel struct {
	LastEducation      bool      `db:"last_education" json:"last_education,omitempty"`
	InstitutionName    string    `db:"institution_name" json:"institution_name,omitempty"`
	EducationLevelUUID uuid.UUID `db:"education_level_uuid" json:"education_level_uuid,omitempty"`
	Major              string    `db:"major" json:"major,omitempty"`
	StartYear          int       `db:"start_year" json:"start_year,omitempty"`
	EndYear            int       `db:"end_year" json:"end_year,omitempty"`
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

type ReadTNSHeaderModelResult struct {
	UUID       uuid.UUID `db:"uuid" json:"uuid"`
	Name       *string   `db:"name" json:"name,omitempty"`
	NIP        *string   `db:"nip" json:"nip,omitempty"`
	Email      *string   `db:"email" json:"email,omitempty"`
	Phone      *string   `db:"phone" json:"phone,omitempty"`
	Occupation string    `db:"occupation" json:"occupation,omitempty"`
	Status     string    `db:"status" json:"status"`
}

type ReadTNSDetailModelResult struct {
	UUID           uuid.UUID         `db:"uuid" json:"uuid"`
	Name           string            `db:"name" json:"name,omitempty"`
	Email          string            `db:"email" json:"email,omitempty"`
	Phone          string            `db:"phone" json:"phone,omitempty"`
	Gender         string            `db:"gender" json:"gender,omitempty"`
	BirthPlace     string            `db:"birth_place" json:"birth_place,omitempty"`
	BirthDate      time.Time         `db:"birth_date" json:"birth_date,omitempty" time_format:"2006-01-02"`
	Address        string            `db:"address" json:"address,omitempty"`
	NIK            string            `db:"nik" json:"nik,omitempty"`
	NUPTK          *string           `db:"nuptk" json:"nuptk,omitempty"`
	NIP            *string           `db:"nip" json:"nip,omitempty"`
	JoinDate       time.Time         `db:"join_date" json:"join_date,omitempty"`
	ResignDate     *time.Time        `db:"resign_date" json:"resign_date,omitempty"`
	StatusUser     string            `db:"status_user" json:"status_user,omitempty"`
	EmployeeStatus string            `db:"employee_status" json:"employee_status,omitempty"`
	Educations     []json.RawMessage `db:"educations" json:"educations,omitempty"`
	Titles         []json.RawMessage `db:"titles" json:"titles,omitempty"`
	Positions      []string          `db:"positions" json:"positions,omitempty"`
	Subject        []string          `db:"subject" json:"subject,omitempty"`
}
