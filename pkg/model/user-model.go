package model

import (
	"time"

	"github.com/google/uuid"
)

type UserModel struct {
	UUID        uuid.UUID  `db:"uuid" json:"uuid"`
	TenantUUID  uuid.UUID  `db:"tenant_uuid" json:"tenant_uuid"`
	Name        *string    `db:"name" json:"name,omitempty"`
	Email       *string    `db:"email" json:"email,omitempty"`
	Phone       *string    `db:"phone" json:"phone,omitempty"`
	Address     *string    `db:"address" json:"address,omitempty"`
	ImgLocation *string    `db:"img_location" json:"img_location,omitempty"`
	RoleUUID    uuid.UUID  `db:"role_uuid" json:"role_uuid"`
	StatusUUID  uuid.UUID  `db:"status_uuid" json:"status_uuid"`
	CreatedDate time.Time  `db:"created_date" json:"created_date"`
	UpdatedDate *time.Time `db:"updated_date" json:"updated_date,omitempty"`
	Version     *string    `db:"version" json:"version,omitempty"`
}
