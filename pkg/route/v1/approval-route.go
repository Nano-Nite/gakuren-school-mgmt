package v1

import (
	"errors"
	"strings"

	"gakuren-system.com/pkg/helper"
	"gakuren-system.com/pkg/model"
	"github.com/gofiber/fiber/v3"
)

func SetupApprovalRoute(app *fiber.App, API_VERSION string) {
	app.Post(API_VERSION+"/school/approval/my-approval", func(c fiber.Ctx) error {
		payload := new(model.SearchPayload)
		// tenantUUID := c.Get("tenant_uuid")
		authHeader := c.Get("Authorization")

		if len(authHeader) == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Bad request", nil, nil)
		}

		//* validate user using access token
		userUUID, err := helper.GetUserUUIDByAccessToken(authHeader)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, err)
		}
		if userUUID == nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, nil)
		}

		//* validate body
		if err := c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}

		// if len(tenantUUID) == 0 {
		// 	return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or Missing between request body and header", nil, nil)
		// }

		selectedUser, err := helper.GetUser(userUUID.String())
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "User not found", nil, err)
		}

		data, dataStat, err := helper.MyApproval(selectedUser.UUID.String(), selectedUser.RoleUUID.String(), *payload)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}

		result := make(map[string]interface{})

		result["data_statistic"] = dataStat
		result["result"] = data

		return helper.ReturnResponse(c, fiber.StatusOK, "success", result, nil)
	})

	app.Get(API_VERSION+"/school/approval/my-approval", func(c fiber.Ctx) error {
		approvalUUID := c.Query("uuid")
		tenantUUID := c.Get("tenant_uuid")
		authHeader := c.Get("Authorization")

		if len(approvalUUID) == 0 || len(tenantUUID) == 0 || len(authHeader) == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Bad request", nil, nil)
		}

		//* validate user using access token
		userUUID, err := helper.GetUserUUIDByAccessToken(authHeader)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, err)
		}
		if userUUID == nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, nil)
		}

		if len(tenantUUID) == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or Missing between request body and header", nil, nil)
		}

		result, err := helper.DetailApproval(approvalUUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, nil)
		}

		return helper.ReturnResponse(c, fiber.StatusOK, "success", result, nil)
	})

	app.Patch(API_VERSION+"/school/approval/my-approval/execute", func(c fiber.Ctx) error {
		payload := new(model.ExecuteApprovalPayload)
		tenantUUID := c.Get("tenant_uuid")
		authHeader := c.Get("Authorization")

		if len(tenantUUID) == 0 || len(authHeader) == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Bad request", nil, nil)
		}

		//* validate user using access token
		userUUID, err := helper.GetUserUUIDByAccessToken(authHeader)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, err)
		}
		if userUUID == nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, nil)
		}

		if len(tenantUUID) == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or Missing between request body and header", nil, nil)
		}

		//* validate body
		if err := c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}

		if payload.Command == "" || payload.UUID == "" {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Bad request", nil, nil)
		}

		//* getting mandatory data
		selectedUser, err := helper.GetUser(userUUID.String())
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, nil)
		}

		selectedApproval, err := helper.DetailApproval(payload.UUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, nil)
		}

		//* pre-data check
		// status check
		if !strings.EqualFold(selectedApproval.DetailApprovalHeader.Status, helper.STATUS_ACTIVE) {
			return helper.ReturnResponse(c, fiber.StatusOK, "Approval already finalize", nil, errors.New("Approval already finalize"))
		}

		command := strings.ToUpper(strings.TrimSpace(payload.Command))
		if command != helper.ACTION_CODE_CANCEL && command != helper.ACTION_CODE_REJECT && command != helper.ACTION_CODE_APPROVE {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Unsupported approval command", nil, nil)
		}
		if (command == helper.ACTION_CODE_CANCEL || command == helper.ACTION_CODE_REJECT) && strings.TrimSpace(payload.Note) == "" {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Note is required", nil, errors.New("missing note"))
		}

		var note *string
		if strings.TrimSpace(payload.Note) != "" {
			note = &payload.Note
		}
		finalized, err := helper.ExecuteApproval(payload.UUID, tenantUUID, selectedUser.UUID, selectedUser.RoleUUID, command, note)
		if err != nil {
			switch {
			case errors.Is(err, helper.ErrApprovalFinalized):
				return helper.ReturnResponse(c, fiber.StatusConflict, "Approval already finalized", nil, err)
			case errors.Is(err, helper.ErrApprovalUnauthorized):
				return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Unauthorized", nil, err)
			case errors.Is(err, helper.ErrApprovalDuplicateAction):
				return helper.ReturnResponse(c, fiber.StatusConflict, "Approval step already actioned", nil, err)
			default:
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
			}
		}

		result := map[string]interface{}{"uuid": payload.UUID, "command": command, "finalized": finalized}

		return helper.ReturnResponse(c, fiber.StatusOK, "success", result, nil)
	})
}
