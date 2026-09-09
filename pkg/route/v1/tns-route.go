package v1

import (
	"time"

	"gakuren-system.com/pkg/helper"
	"gakuren-system.com/pkg/model"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func SetupTNSRoute(app *fiber.App, apiVersion string) {
	baseURL := apiVersion + "/school/tns"

	app.Post(baseURL+"/create", func(c fiber.Ctx) error {
		payload := new(model.CreateTNSModel)
		schoolUUID, tenantUUID, requesterUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}
		if err = c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body", nil, err)
		}

		if ok, permissionErr := helper.GetUserPermission(requesterUUID.String(), helper.CREATE_TNS_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		// check user existing
		if err = helper.UserTNSValidity(*payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "User contain another user data", nil, err)
		}

		// get teacher or staff role
		var selectedRole *model.RoleModel
		if payload.IsStaff {
			selectedRole, err = helper.GetRoleByAbbrName(helper.ROLE_STAFF)
		} else {
			selectedRole, err = helper.GetRoleByAbbrName(helper.ROLE_TEACHER)
		}
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to get role", nil, err)
		}

		data := model.UserModel{
			TenantUUID:  tenantUUID,
			Name:        &payload.Biodata.Fullname,
			Email:       &payload.Biodata.Email,
			Phone:       &payload.Biodata.Phone,
			Address:     &payload.Biodata.Address,
			ImgLocation: &payload.ImgLocation,
			StatusUUID:  helper.DB_UUID_STATUS_NEWUSER,
			RoleUUID:    selectedRole.UUID,
			CreatedDate: time.Now(),
		}
		canBypass, err := helper.ValidateApprovalBypass(requesterUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to check approval bypass", nil, err)
		}
		if canBypass {
			id, err := helper.InsertTNS(*payload, data, tenantUUID, schoolUUID)
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create Teacher or Staff", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": id}, nil)
		}
		workflow, err := helper.DetermineWorkflow(schoolUUID, tenantUUID, requesterUUID, helper.CREATE_TNS_PERMISSION, helper.ACTION_CODE_CREATE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, err)
		}
		if workflow == nil {
			var id *uuid.UUID
			err = helper.ExecuteWorkflowFallback(func() error {
				// create user data first then employee
				id, err = helper.InsertTNS(*payload, data, tenantUUID, schoolUUID)
				return err
			})
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Create rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": id}, nil)
		}
		approvalUUID, err := helper.CreateApproval(*workflow, schoolUUID, tenantUUID, requesterUUID, nil, helper.ACTION_CODE_CREATE, helper.TNS_ENTITY_TYPE, helper.TNS_MODULE_CODE, payload)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user approval", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"approval_uuid": approvalUUID}, nil)
	})

	app.Patch(baseURL+"/update", func(c fiber.Ctx) error {
		payload := new(model.UpdateTNSModel)

		// validation header
		schoolUUID, tenantUUID, requesterUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}

		// body check
		if err = c.Bind().Body(payload); err != nil || payload.UUID == uuid.Nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body, uuid is required", nil, err)
		}

		// permission check
		if ok, permissionErr := helper.GetUserPermission(requesterUUID.String(), helper.UPDATE_TNS_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		// get data
		selectedData, err := helper.GetTNS(schoolUUID, tenantUUID, payload.UUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Teacher or staff not found", nil, err)
		}

		// pending data check
		if selectedData.StatusUser != helper.DB_UUID_STATUS_NEWUSER {
			if selectedData.StatusUser != helper.DB_UUID_STATUS_ACTIVE && selectedData.StatusUser != helper.DB_UUID_STATUS_INACTIVE {
				return helper.ReturnResponse(c, fiber.StatusConflict, "Update rejected because user has a pending action", nil, nil)
			}
		}

		// activate case
		activate := selectedData.StatusUser == helper.DB_UUID_STATUS_INACTIVE && *payload.StatusUserUUID == helper.DB_UUID_STATUS_ACTIVE && payload.Activate
		if activate {
			*payload.StatusUserUUID = helper.DB_UUID_STATUS_ACTIVE
		}

		// bypass permission check
		canBypass, err := helper.ValidateApprovalBypass(requesterUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to check approval bypass", nil, err)
		}
		operation := func() error {
			err = helper.UpdateTNSStatus(*payload, schoolUUID, tenantUUID, *payload.StatusUserUUID)
			if !activate {
				return helper.UpdateTNS(*payload, tenantUUID, schoolUUID)
			}
			return err
		}
		if canBypass {
			if err = operation(); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to update user", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		// workflow check
		workflow, err := helper.DetermineWorkflow(schoolUUID, tenantUUID, requesterUUID, helper.UPDATE_TNS_PERMISSION, helper.ACTION_CODE_UPDATE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, err)
		}
		if workflow == nil {
			if err = helper.ExecuteWorkflowFallback(operation); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Update rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		// create approval
		approvalUUID, err := helper.CreateApproval(*workflow, schoolUUID, tenantUUID, requesterUUID, &payload.UUID, helper.ACTION_CODE_UPDATE, helper.TNS_ENTITY_TYPE, helper.TNS_MODULE_CODE, payload)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user update approval", nil, err)
		}

		// update user status to Pending
		if err = helper.UpdateTNSStatus(*payload, schoolUUID, tenantUUID, helper.DB_UUID_STATUS_PENDING); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to mark user pending", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": payload.UUID, "approval_uuid": approvalUUID}, nil)
	})

	app.Delete(baseURL+"/delete", func(c fiber.Ctx) error {
		payload := new(model.DeleteTNSModel)

		// validation header
		schoolUUID, tenantUUID, requesterUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}

		// body check
		if err = c.Bind().Body(payload); err != nil || payload.UserUUID == uuid.Nil || payload.EmployeeUUID == uuid.Nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body", nil, err)
		}

		// permission check
		if ok, permissionErr := helper.GetUserPermission(requesterUUID.String(), helper.DELETE_TNS_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		// get data
		selectedData, err := helper.GetTNS(schoolUUID, tenantUUID, payload.UserUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Teacher or staff not found", nil, err)
		}

		// status validation
		if selectedData.StatusUser != helper.DB_UUID_STATUS_NEWUSER {
			if selectedData.StatusUser != helper.DB_UUID_STATUS_ACTIVE {
				return helper.ReturnResponse(c, fiber.StatusConflict, "Delete rejected because user is not active", nil, nil)
			}
		}

		selectedDataModel := model.UpdateTNSModel{
			UUID:         selectedData.UUID,
			EmployeeUUID: selectedData.EmployeeUUID,
		}

		// bypass check
		canBypass, err := helper.ValidateApprovalBypass(requesterUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to check approval bypass", nil, err)
		}
		operation := func() error {
			return helper.SoftDeleteTNS(selectedDataModel, schoolUUID, tenantUUID)
		}
		if canBypass {
			if err = operation(); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to delete user", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		workflow, err := helper.DetermineWorkflow(schoolUUID, tenantUUID, requesterUUID, helper.DELETE_TNS_PERMISSION, helper.ACTION_CODE_DELETE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, err)
		}
		if workflow == nil {
			if err = helper.ExecuteWorkflowFallback(operation); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Delete rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		approvalUUID, err := helper.CreateApproval(*workflow, schoolUUID, tenantUUID, requesterUUID, &payload.UserUUID, helper.ACTION_CODE_DELETE, helper.TNS_ENTITY_TYPE, helper.TNS_MODULE_CODE, selectedData)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user delete approval", nil, err)
		}

		// update status to Pending
		if err = helper.UpdateTNSStatus(selectedDataModel, schoolUUID, tenantUUID, helper.DB_UUID_STATUS_PENDING); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to mark user pending", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": payload.UserUUID, "approval_uuid": approvalUUID}, nil)
	})

	app.Post(baseURL+"/get", func(c fiber.Ctx) error {
		payload := new(model.SearchPayload)
		schoolUUID, tenantUUID, userUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}
		if err = c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}
		data, stats, err := helper.SearchTNSHeader(schoolUUID, tenantUUID, userUUID, *payload)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Failed to search users", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"data_statistic": stats, "result": data}, nil)
	})

	app.Get(baseURL+"/get", func(c fiber.Ctx) error {
		tnsUUIDQuery := c.Query("uuid")

		tnsUUID, err := uuid.Parse(tnsUUIDQuery)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}

		schoolUUID, tenantUUID, userUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}

		result, err := helper.SearchTNSDetail(schoolUUID, tenantUUID, userUUID, tnsUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Failed to search users", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", result, nil)
	})

	app.Get(baseURL+"/get-homeroom-teacher", func(c fiber.Ctx) error {
		schoolUUID, tenantUUID, _, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}

		result, err := helper.GetHomeroomTeacher(schoolUUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Failed to search users", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", result, nil)
	})

}
