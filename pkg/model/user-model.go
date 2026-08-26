package model

import (
	"time"

	"github.com/google/uuid"
)

type UserModel struct {
	UUID        uuid.UUID  `db:"uuid"`
	TenantUUID  uuid.UUID  `db:"tenant_uuid"`
	Name        string     `db:"name"`
	Email       string     `db:"email"`
	Phone       string     `db:"phone"`
	Address     string     `db:"address"`
	ImgLocation *string    `db:"img_location"`
	RoleUUID    uuid.UUID  `db:"role_uuid"`
	StatusUUID  uuid.UUID  `db:"status_uuid"`
	CreatedDate time.Time  `db:"created_date"`
	UpdatedDate *time.Time `db:"updated_date"`
	Version     string     `db:"version"`
}
