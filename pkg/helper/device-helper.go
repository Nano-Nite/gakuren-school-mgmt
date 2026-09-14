package helper

import (
	"context"
	"errors"
	"fmt"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func ValidateRegisterPayloadToDB(data model.RegisterDevicePayload, schoolUUID, tenantUUID uuid.UUID) ([]model.TrustedDeviceModel, error) {
	query := `
	select *
	from attendance_sch.trusted_device
	where tenant_uuid = $1 
	and school_uuid = $2
	and location_uuid = $3
	and device_name = $4
	and status = $5
	`
	selectedDetail, err := db.GetMultipleDataByQuery[model.TrustedDeviceModel](query, tenantUUID, schoolUUID, data.LocationUUID, data.DeviceName, model.DEVICE_STATUS_ACTIVE)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return nil, err
		}
	}
	if selectedDetail == nil {
		return nil, errors.New("Fail to get detail data")
	}

	return *selectedDetail, nil
}

func InsertTrustedDevice(data model.TrustedDeviceModel, tx pgx.Tx, ctx context.Context) (uuid.UUID, error) {
	query := `
		INSERT INTO attendance_sch.trusted_device (
			tenant_uuid, school_uuid, location_uuid, device_code, device_name, device_identifier, status, registered_by, registered_at, created_date
		) VALUES (
		 	$1, $2, $3, $4, $5, $6, $7, $8, now(), now()
		)
		RETURNING uuid;
	`
	var trustedDeviceUUID uuid.UUID
	err := tx.QueryRow(ctx, query,
		data.TenantUUID,
		data.SchoolUUID,
		data.LocationUUID,
		data.DeviceCode,
		data.DeviceName,
		data.DeviceIdentifier,
		data.Status,
		data.RegisteredBy,
	).Scan(&trustedDeviceUUID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert user: %w", err)
	}

	return trustedDeviceUUID, nil
}

func InsertTrustedDeviceKey(data model.TrustedDeviceKeyModel, tx pgx.Tx, ctx context.Context) error {
	query := `
		INSERT INTO attendance_sch.trusted_device_key (
			tenant_uuid, school_uuid, trusted_device_uuid, key_version, algorithm, public_key, public_key_format, fingerprint, status, valid_from, created_date
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now()
		)
		returning uuid;
	`
	var trustedDeviceUUID uuid.UUID
	err := tx.QueryRow(ctx, query,
		data.TenantUUID,
		data.SchoolUUID,
		data.TrustedDeviceUUID,
		data.KeyVersion,
		data.Algorithm,
		data.PublicKey,
		data.PublicKeyFormat,
		data.FingerPrint,
		data.Status,
	).Scan(&trustedDeviceUUID)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}
