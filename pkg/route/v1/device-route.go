package v1

import (
	"context"
	"strings"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/helper"
	"gakuren-system.com/pkg/model"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func SetupDeviceRoute(app *fiber.App, API_VERSION string) {
	app.Post(API_VERSION+"/school/trusted-device/register", func(c fiber.Ctx) error {
		payload := new(model.RegisterDevicePayload)

		//* validate JWT
		schoolUUID, tenantUUID, requesterUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}

		if err = c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body", nil, err)
		}

		//* validate permission
		if ok, permissionErr := helper.GetUserPermission(requesterUUID.String(), helper.CREATE_SETTING_DEVICE_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		//* validate data to db
		validateDatas, err := helper.ValidateRegisterPayloadToDB(*payload, schoolUUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to get trusted device data", nil, err)
		}

		if len(validateDatas) > 0 {
			return helper.ReturnResponse(c, fiber.StatusConflict, "Multiple data detected", nil, err)
		}

		//* convert payload into disire model
		dataTrustedDevice := model.TrustedDeviceModel{
			TenantUUID:       tenantUUID,
			SchoolUUID:       schoolUUID,
			LocationUUID:     payload.LocationUUID,
			DeviceCode:       strings.ToUpper(strings.ReplaceAll(payload.DeviceName, " ", "-")),
			DeviceName:       payload.DeviceName,
			DeviceIdentifier: payload.DeviceIdentifier.String(),
			Status:           model.DEVICE_STATUS_PENDING,
			RegisteredBy:     requesterUUID,
		}

		dataTrustedDeviceKey := model.TrustedDeviceKeyModel{
			TenantUUID:        tenantUUID,
			SchoolUUID:        schoolUUID,
			TrustedDeviceUUID: uuid.Nil,
			KeyVersion:        1,
			Algorithm:         payload.Key["algorithm"].(string),
			PublicKey:         payload.Key["public_key"].(string),
			PublicKeyFormat:   payload.Key["public_key_format"].(string),
			FingerPrint:       payload.Key["fingerprint"].(string),
			Status:            model.DEVICE_KEY_STATUS_ACTIVE,
		}

		//* bypass approval check
		canBypass, err := helper.ValidateApprovalBypass(requesterUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to check approval bypass", nil, err)
		}
		if canBypass {
			tx, err := db.Conn.Begin(context.Background())
			if err != nil {
				return err
			}
			defer tx.Rollback(context.Background())

			// pre-set trusted device status due user can bypass approval
			dataTrustedDevice.Status = model.DEVICE_STATUS_ACTIVE

			trustedDeviceID, err := helper.InsertTrustedDevice(dataTrustedDevice, tx, c.Context())
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create Trusted Device", nil, err)
			}

			// set trusted device uuid using previous query
			dataTrustedDeviceKey.TrustedDeviceUUID = trustedDeviceID

			err = helper.InsertTrustedDeviceKey(dataTrustedDeviceKey, tx, c.Context())
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create Trusted Device Key", nil, err)
			}

			// commit
			if tx.Commit(c.Context()) != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to commit Trusted Device", nil, nil)
			}

			result := make(map[string]interface{})
			result["device_uuid"] = trustedDeviceID
			result["school_uuid"] = schoolUUID
			result["location_uuid"] = dataTrustedDevice.LocationUUID
			result["trusted"] = true

			return helper.ReturnResponse(c, fiber.StatusOK, "success", result, nil)
		}

		return helper.ReturnResponse(c, fiber.StatusOK, "success", nil, nil)
	})
}
