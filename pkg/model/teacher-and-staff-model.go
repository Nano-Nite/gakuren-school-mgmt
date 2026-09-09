package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TNSModel struct {
	UUID           uuid.UUID         `db:"uuid" json:"uuid"`
	EmployeeUUID   uuid.UUID         `db:"employee_uuid" json:"employee_uuid"`
	Name           string            `db:"name" json:"name,omitempty"`
	Email          string            `db:"email" json:"email,omitempty"`
	Phone          string            `db:"phone" json:"phone,omitempty"`
	Gender         uuid.UUID         `db:"gender" json:"gender,omitempty"`
	BirthPlace     string            `db:"birth_place" json:"birth_place,omitempty"`
	BirthDate      time.Time         `db:"birth_date" json:"birth_date,omitempty" time_format:"2006-01-02"`
	Address        string            `db:"address" json:"address,omitempty"`
	NIK            string            `db:"nik" json:"nik,omitempty"`
	NUPTK          *string           `db:"nuptk" json:"nuptk,omitempty"`
	NIP            *string           `db:"nip" json:"nip,omitempty"`
	JoinDate       time.Time         `db:"join_date" json:"join_date,omitempty"`
	ResignDate     *time.Time        `db:"resign_date" json:"resign_date,omitempty"`
	StatusUser     uuid.UUID         `db:"status_user" json:"status_user,omitempty"`
	EmployeeStatus uuid.UUID         `db:"employee_status" json:"employee_status,omitempty"`
	Educations     []json.RawMessage `db:"educations" json:"educations,omitempty"`
	Titles         []json.RawMessage `db:"titles" json:"titles,omitempty"`
	Positions      []string          `db:"positions" json:"positions,omitempty"`
	Subject        []string          `db:"subject" json:"subject,omitempty"`
}

type CreateTNSModel struct {
	Biodata            TNSBioModel           `db:"biodata" json:"biodata,omitempty"`
	EducationLevel     []TNSEducationModel   `db:"education_level" json:"education_level,omitempty"`
	ImgLocation        string                `db:"img_location" json:"img_location,omitempty"`
	IsStaff            bool                  `db:"is_staff" json:"is_staff,omitempty"`
	NIK                string                `db:"nik" json:"nik,omitempty"`
	NUPTK              *string               `db:"nuptk" json:"nuptk,omitempty"`
	NIP                *string               `db:"nip" json:"nip,omitempty"`
	Positions          []ResultPositionModel `db:"positions" json:"positions,omitempty"`
	Titles             []ResultTitleModel    `db:"titles" json:"titles,omitempty"`
	Subjects           []ResultSubjectModel  `db:"subjects" json:"subjects,omitempty"`
	JoinDate           *time.Time            `db:"join_date" json:"join_date,omitempty"`
	ResignDate         *time.Time            `db:"resign_date" json:"resign_date,omitempty"`
	EmployeeStatusUUID uuid.UUID             `db:"employee_status_uuid" json:"employee_status_uuid,omitempty"`
	CreatedDate        time.Time             `db:"created_date" json:"created_date,omitempty"`
	UpdatedDate        *time.Time            `db:"updated_date" json:"updated_date,omitempty"`
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

type UpdateTNSModel struct {
	UUID               uuid.UUID             `db:"uuid" json:"uuid"`
	EmployeeUUID       uuid.UUID             `db:"employee_uuid" json:"employee_uuid"`
	Biodata            TNSBioModel           `db:"biodata" json:"biodata,omitempty"`
	EducationLevel     []TNSEducationModel   `db:"education_level" json:"education_level,omitempty"`
	ImgLocation        string                `db:"img_location" json:"img_location,omitempty"`
	IsStaff            bool                  `db:"is_staff" json:"is_staff,omitempty"`
	NIK                string                `db:"nik" json:"nik,omitempty"`
	NUPTK              *string               `db:"nuptk" json:"nuptk,omitempty"`
	NIP                *string               `db:"nip" json:"nip,omitempty"`
	Positions          []ResultPositionModel `db:"positions" json:"positions,omitempty"`
	Titles             []ResultTitleModel    `db:"titles" json:"titles,omitempty"`
	Subjects           []ResultSubjectModel  `db:"subjects" json:"subjects,omitempty"`
	JoinDate           *time.Time            `db:"join_date" json:"join_date,omitempty"`
	ResignDate         *time.Time            `db:"resign_date" json:"resign_date,omitempty"`
	CreatedDate        time.Time             `db:"created_date" json:"created_date,omitempty"`
	UpdatedDate        *time.Time            `db:"updated_date" json:"updated_date,omitempty"`
	StatusUserUUID     *uuid.UUID            `db:"status_user" json:"status_user,omitempty"`
	EmployeeStatusUUID uuid.UUID             `db:"employee_status" json:"employee_status,omitempty"`
	Activate           bool                  `db:"activate" json:"activate,omitempty"`
}

type DeleteTNSModel struct {
	UserUUID     uuid.UUID `json:"user_uuid"`
	EmployeeUUID uuid.UUID `json:"employee_uuid"`
}

type ActivateTNSModel struct {
	UserUUID     uuid.UUID `json:"user_uuid"`
	EmployeeUUID uuid.UUID `json:"employee_uuid"`
	Activate     bool      `json:"activate"`
}

type ReadTNSHeaderModelResult struct {
	UUID           uuid.UUID `db:"uuid" json:"uuid"`
	Name           *string   `db:"name" json:"name,omitempty"`
	NIP            *string   `db:"nip" json:"nip,omitempty"`
	Email          *string   `db:"email" json:"email,omitempty"`
	Phone          *string   `db:"phone" json:"phone,omitempty"`
	Occupation     string    `db:"occupation" json:"occupation,omitempty"`
	StatusUser     string    `db:"status_user" json:"status_user,omitempty"`
	EmployeeStatus string    `db:"employee_status" json:"employee_status,omitempty"`
}

type ReadTNSDetailModelResult struct {
	UUID           uuid.UUID         `db:"uuid" json:"uuid"`
	EmployeeUUID   uuid.UUID         `db:"employee_uuid" json:"employee_uuid"`
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
