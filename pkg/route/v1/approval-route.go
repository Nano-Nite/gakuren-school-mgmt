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

		selectedUser, err := helper.GetUser(userUUID.String())
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, nil)
		}

		selectedRole, err := helper.GetRole(selectedUser.RoleUUID.String())
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, nil)
		}

		selectedApproval, err := helper.DetailApproval(payload.UUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, nil)
		}

		//* handle CANCEL approval
		if strings.EqualFold(payload.Command, helper.ACTION_CODE_CANCEL) {
			if selectedApproval.DetailApprovalHeader.CurrentStep != 1 {
				return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Unauthorized", nil, nil)
			} else {
				// canceling approval make notes as mandatory
				if payload.Note == "" || len(payload.Note) == 0 {
					return helper.ReturnResponse(c, fiber.StatusBadRequest, "Bad request", nil, errors.New("Missing notes"))
				}

				// status check
				if !strings.EqualFold(selectedApproval.DetailApprovalHeader.Status, helper.STATUS_ACTIVE) {
					return helper.ReturnResponse(c, fiber.StatusOK, "Approval already finalize", nil, errors.New("Approval already finalize"))
				}

				// maker check
				if !strings.EqualFold(selectedApproval.DetailApprovalHeader.RequestedBy, selectedUser.Name) {
					return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Unauthorized", nil, nil)
				}

				// role check
				if !strings.EqualFold(selectedApproval.DetailApprovalHeader.RoleName, selectedRole.Name) {
					return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Unauthorized", nil, nil)
				}

				//* Update approval instance status
				if err = helper.UpdateApprovalInstancteStatus(payload.UUID, helper.ACTION_CODE_CANCEL); err != nil {
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
				}

				//* insert approval action -> Cancel
				var approvalActionPayload model.ApprovalAction
				approvalActionPayload.ApprovalInstanceUUID = selectedApproval.DetailApprovalHeader.UUID
				approvalActionPayload.ApprovalStepUUID = nil
				approvalActionPayload.ActionCode = helper.ACTION_CODE_CANCEL
				approvalActionPayload.ActedBy = selectedUser.UUID
				approvalActionPayload.Note = &payload.Note

				if err = helper.InsertApprovalAction(approvalActionPayload); err != nil {
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
				}

			}
		}

		return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
	})
}
