package model

import (
	"time"

	"github.com/google/uuid"
)

// (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'DISABLED'::character varying, 'REVOKED'::character varying])::text[])))
const DEVICE_STATUS_ACTIVE = "ACTIVE"
const DEVICE_STATUS_PENDING = "PENDING"
const DEVICE_STATUS_DISABLED = "DISABLED"
const DEVICE_STATUS_REVOKED = "REVOKED"

// (((status)::text = ANY ((ARRAY['ACTIVE'::character varying, 'ROTATED'::character varying, 'REVOKED'::character varying])::text[])))
const DEVICE_KEY_STATUS_ACTIVE = "ACTIVE"
const DEVICE_KEY_STATUS_ROTATED = "ROTATED"
const DEVICE_KEY_STATUS_REVOKED = "REVOKED"

type TrustedDeviceModel struct {
	UUID             uuid.UUID `db:"uuid" json:"uuid"`
	TenantUUID       uuid.UUID `db:"tenant_uuid" json:"tenant_uuid"`
	SchoolUUID       uuid.UUID `db:"school_uuid" json:"school_uuid"`
	LocationUUID     uuid.UUID `db:"location_uuid" json:"location_uuid"`
	DeviceCode       string    `db:"device_code" json:"device_code"`
	DeviceName       string    `db:"device_name" json:"device_name"`
	DeviceIdentifier string    `db:"device_identifier" json:"device_identifier"`
	Status           string    `db:"status" json:"status"`
	RegisteredBy     uuid.UUID `db:"registered_by" json:"registered_by"`
	RegisteredAt     time.Time `db:"registered_at" json:"registered_at"`
	LastSeenAt       time.Time `db:"last_seen_at" json:"last_seen_at"`
	LastSyncAt       time.Time `db:"last_sync_at" json:"last_sync_at"`
	RevokedBy        uuid.UUID `db:"revoked_by" json:"revoked_by"`
	RevokedAt        time.Time `db:"revoked_at" json:"revoked_at"`
	CreatedDate      time.Time `db:"created_date" json:"created_date"`
	UpdatedDate      time.Time `db:"updated_date" json:"updated_date"`
	ActivatedAt      time.Time `db:"activated_at" json:"activated_at"`
	ActivatedBy      uuid.UUID `db:"activated_by" json:"activated_by"`
}

type TrustedDeviceKeyModel struct {
	UUID              uuid.UUID `db:"uuid" json:"uuid"`
	TenantUUID        uuid.UUID `db:"tenant_uuid" json:"tenant_uuid"`
	SchoolUUID        uuid.UUID `db:"school_uuid" json:"school_uuid"`
	TrustedDeviceUUID uuid.UUID `db:"trusted_device_uuid" json:"trusted_device_uuid"`
	KeyVersion        int       `db:"key_version" json:"key_version"`
	Algorithm         string    `db:"algorithm" json:"algorithm"`
	PublicKey         string    `db:"public_key" json:"public_key"`
	PublicKeyFormat   string    `db:"public_key_format" json:"public_key_format"`
	FingerPrint       string    `db:"fingerprint" json:"fingerprint"`
	Status            string    `db:"status" json:"status"`
	ValidFrom         time.Time `db:"valid_from" json:"valid_from"`
	ValidUntil        time.Time `db:"valid_until" json:"valid_until"`
	RevokedAt         time.Time `db:"revoked_at" json:"revoked_at"`
	RevokedBy         uuid.UUID `db:"revoked_by" json:"revoked_by"`
	CreatedDate       time.Time `db:"created_date" json:"created_date"`
}

type RegisterDevicePayload struct {
	DeviceName        string                 `json:"device_name"`
	LocationUUID      uuid.UUID              `json:"location_uuid"`
	DeviceIdentifier  uuid.UUID              `json:"device_identifier"`
	Key               map[string]interface{} `json:"key"`
	OfflineCapability map[string]interface{} `json:"offline_capability"`
}
