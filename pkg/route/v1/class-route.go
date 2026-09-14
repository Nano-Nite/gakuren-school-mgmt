package v1

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gakuren-system.com/pkg/helper"
	"gakuren-system.com/pkg/model"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func SetupClassRoute(app *fiber.App, API_VERSION string) {
	classBaseURL := API_VERSION + "/school/class"

	// create
	app.Post(classBaseURL+"/create", func(c fiber.Ctx) error {
		payload := new(model.CreateClassModel)

		//* validate user using access token
		schoolUUID, tenantUUID, userUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, err)
		}

		//* validate body
		if err := c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}

		//* validate user permission
		ok, err := helper.GetUserPermission(userUUID.String(), helper.CREATE_CLASS_PERMISSION)
		if err != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, nil)
		}

		//* is user have bypass permission
		canBypass, err := helper.ApprovalBypass(userUUID.String())
		if err != nil {
			if err.Error() != "no rows in result set" {
				return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, nil)
			}
		}

		//* validate payload
		if len(payload.Name) == 0 || payload.Level == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or Missing between request body and header", nil, nil)
		}

		//* validate duplicated data
		exists, err := helper.CheckClassExists(*payload, schoolUUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
		}
		if exists {
			return helper.ReturnResponse(c, fiber.StatusConflict, "Class already exists", nil, nil)
		}

		var insertData model.ClassModel
		insertData.UUID = nil
		insertData.Name = payload.Name
		if payload.AbbrName != nil && len(*payload.AbbrName) > 0 {
			insertData.AbbrName = payload.AbbrName
		}
		insertData.Level = payload.Level
		insertData.HomeroomTeacher = payload.HomeroomTeacher
		insertData.StatusUUID = helper.DB_UUID_STATUS_ACTIVE
		insertData.CreatedDate = time.Now()
		insertData.UpdatedDate = nil
		insertData.TenantUUID = tenantUUID
		insertData.SchoolUUID = schoolUUID

		if canBypass { // if user can bypass approval
			if err = helper.InsertClass(insertData, schoolUUID, tenantUUID); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
			}
		} else { // if user cannot bypass approval
			//* workflow approval logic
			selectedWorkflow, err := helper.DetermineWorkflowApproval(schoolUUID.String(), tenantUUID.String(), userUUID.String(), helper.CREATE_CLASS_PERMISSION, helper.ACTION_CODE_CREATE, helper.DB_UUID_STATUS_ACTIVE.String())
			if err != nil {
				if err.Error() != "no rows in result set" {
					return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, nil)
				}
			}

			//* action when workflow not found
			if selectedWorkflow == nil {
				selectedAction, err := helper.GetVariableUsingKey(helper.WORKFLOW_NOTFOUND_BEHAVIOUR)
				if err != nil {
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
				}

				if helper.REJECT == selectedAction.Value { // when action must REJECT
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, errors.New("Rejected due workflow behaviour"))
				} else if helper.SAVE == selectedAction.Value { // when action must SAVE
					if err = helper.InsertClass(insertData, schoolUUID, tenantUUID); err != nil {
						return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
					}
				} else { // skip when not reject nor save
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, errors.New("Skipped due workflow behaviour"))
				}
			} else {
				//* create approval workflow logic
				// convert payload to json
				payloadJson, _ := helper.ConvertModelToJSON(insertData)

				// init approval instance
				var instance model.ApprovalInstance
				instance.ApprovalWorkflowUUID = selectedWorkflow.UUID
				instance.SchoolUUID = schoolUUID
				instance.TenantUUID = tenantUUID
				instance.EntityType = helper.CLASS_ENTITY_TYPE
				instance.EntityUUID = nil
				instance.ActionCode = helper.ACTION_CODE_CREATE
				instance.RequestData = json.RawMessage(payloadJson)
				instance.StatusUUID = helper.DB_UUID_STATUS_ACTIVE
				instance.RequestedBy = userUUID
				instance.FinalizedBy = nil
				instance.FinalizedDate = nil
				instance.UpdatedDate = nil

				// insert approval instance
				instanceUUID, err := helper.CreateApprovalInstance(instance, helper.CLASS_MODULE_CODE)
				if err != nil {
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
				}

				// init approval action
				var action model.ApprovalAction
				action.ApprovalInstanceUUID = *instanceUUID
				action.ApprovalStepUUID = nil
				action.ActionCode = helper.ACTION_CODE_SUBMIT
				action.ActedBy = userUUID
				action.Note = nil
				action.CreatedDate = time.Now()

				// insert approval action
				_, err = helper.CreateApprovalAction(action)
				if err != nil {
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
				}
			}
		}

		return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
	})

	// update
	app.Patch(classBaseURL+"/update", func(c fiber.Ctx) error {
		payload := new(model.UpdateClassModel)
		schoolUUID, tenantUUID, userUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}
		if err = c.Bind().Body(payload); err != nil || payload.UUID == uuid.Nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}
		if payload.Name == "" || payload.Level == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Name and level are required", nil, nil)
		}
		if ok, permissionErr := helper.GetUserPermission(userUUID.String(), helper.UPDATE_CLASS_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		// get data
		classData, err := helper.GetClass(payload.UUID, tenantUUID, schoolUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Class not found", nil, err)
		}

		// status check
		if classData.StatusUUID == helper.DB_UUID_STATUS_PENDING || classData.StatusUUID == helper.DB_UUID_STATUS_INACTIVE {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Update rejected due current status is pending", nil, err)
		}

		if classData.StatusUUID == helper.DB_UUID_STATUS_INACTIVE && strings.EqualFold(payload.Status, helper.STATUS_ACTIVE) { // activate case
			classData.StatusUUID = helper.DB_UUID_STATUS_ACTIVE
			classData.TenantUUID = tenantUUID
			classData.SchoolUUID = schoolUUID
		} else {
			classData.Name = payload.Name
			classData.AbbrName = payload.AbbrName
			classData.Level = payload.Level
			classData.HomeroomTeacher = payload.HomeroomTeacher
			classData.TenantUUID = tenantUUID
			classData.SchoolUUID = schoolUUID
		}

		// bypass check
		canBypass, bypassErr := helper.ApprovalBypass(userUUID.String())
		if bypassErr != nil && bypassErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, bypassErr)
		}
		if canBypass {
			if err = helper.UpdateClass(*classData, schoolUUID, tenantUUID); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to update class", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		workflow, workflowErr := helper.DetermineWorkflowApproval(schoolUUID.String(), tenantUUID.String(), userUUID.String(), helper.UPDATE_CLASS_PERMISSION, helper.ACTION_CODE_UPDATE, helper.DB_UUID_STATUS_ACTIVE.String())
		if workflowErr != nil && workflowErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, workflowErr)
		}
		if workflow == nil {
			if err = helper.ExecuteWorkflowFallback(func() error { return helper.UpdateClass(*classData, schoolUUID, tenantUUID) }); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Update rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		instanceUUID, err := helper.CreateApproval(*workflow, schoolUUID, tenantUUID, userUUID, &payload.UUID, helper.ACTION_CODE_UPDATE, helper.CLASS_ENTITY_TYPE, helper.CLASS_MODULE_CODE, classData)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user approval", nil, err)
		}

		// update status data
		classData.StatusUUID = helper.DB_UUID_STATUS_PENDING
		if err = helper.UpdateClassStatus(*classData, classData.StatusUUID, tenantUUID, schoolUUID); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed make status inactive", nil, err)
		}

		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": payload.UUID, "approval_uuid": instanceUUID}, nil)
	})

	// delete
	app.Delete(classBaseURL+"/delete", func(c fiber.Ctx) error {
		payload := new(model.DeleteClassModel)
		schoolUUID, tenantUUID, userUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}
		if err = c.Bind().Body(payload); err != nil || payload.UUID == uuid.Nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}
		if ok, permissionErr := helper.GetUserPermission(userUUID.String(), helper.DELETE_CLASS_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		classData, err := helper.GetClass(payload.UUID, tenantUUID, schoolUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Class not found", nil, err)
		}

		// status check
		if classData.StatusUUID == helper.DB_UUID_STATUS_PENDING || classData.StatusUUID == helper.DB_UUID_STATUS_INACTIVE {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Delete rejected due current status is pending/inactive", nil, err)
		}

		selectedClass, err := helper.GetClass(payload.UUID, tenantUUID, schoolUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Class not found", nil, err)
		}

		canBypass, bypassErr := helper.ApprovalBypass(userUUID.String())
		if bypassErr != nil && bypassErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, bypassErr)
		}
		if canBypass {
			if err = helper.SoftDeleteClass(*selectedClass, tenantUUID, schoolUUID); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to delete class", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		workflow, workflowErr := helper.DetermineWorkflowApproval(schoolUUID.String(), tenantUUID.String(), userUUID.String(), helper.DELETE_CLASS_PERMISSION, helper.ACTION_CODE_DELETE, helper.DB_UUID_STATUS_ACTIVE.String())
		if workflowErr != nil && workflowErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, workflowErr)
		}
		if workflow == nil {
			if err = helper.ExecuteWorkflowFallback(func() error {
				return helper.SoftDeleteClass(*selectedClass, tenantUUID, schoolUUID)
			}); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Delete rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		instanceUUID, err := helper.CreateApproval(*workflow, schoolUUID, tenantUUID, userUUID, &payload.UUID, helper.ACTION_CODE_DELETE, helper.CLASS_ENTITY_TYPE, helper.CLASS_MODULE_CODE, classData)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create approval", nil, err)
		}

		// update status data
		classData.StatusUUID = helper.DB_UUID_STATUS_PENDING
		if err = helper.UpdateClassStatus(*selectedClass, classData.StatusUUID, tenantUUID, schoolUUID); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed make status pending", nil, err)
		}

		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": payload.UUID, "approval_uuid": instanceUUID}, nil)
	})

	// actvate
	app.Patch(classBaseURL+"/activate", func(c fiber.Ctx) error {
		payload := new(model.ActivateClassModel)
		schoolUUID, tenantUUID, userUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}
		if err = c.Bind().Body(payload); err != nil || payload.UUID == uuid.Nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}
		if payload.UUID == uuid.Nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Class UUID is required", nil, nil)
		}
		if ok, permissionErr := helper.GetUserPermission(userUUID.String(), helper.UPDATE_CLASS_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		// get data
		classData, err := helper.GetClass(payload.UUID, tenantUUID, schoolUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Class not found", nil, err)
		}

		// status check
		if classData.StatusUUID == helper.DB_UUID_STATUS_PENDING && classData.StatusUUID != helper.DB_UUID_STATUS_INACTIVE {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Update rejected due current status is pending", nil, err)
		}

		// bypass check
		canBypass, bypassErr := helper.ApprovalBypass(userUUID.String())
		if bypassErr != nil && bypassErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, bypassErr)
		}
		if canBypass {
			if err = helper.UpdateClassStatus(*classData, helper.DB_UUID_STATUS_ACTIVE, tenantUUID, schoolUUID); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to update class", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		workflow, workflowErr := helper.DetermineWorkflowApproval(schoolUUID.String(), tenantUUID.String(), userUUID.String(), helper.UPDATE_CLASS_PERMISSION, helper.ACTION_CODE_UPDATE, helper.DB_UUID_STATUS_ACTIVE.String())
		if workflowErr != nil && workflowErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, workflowErr)
		}
		if workflow == nil {
			if err = helper.ExecuteWorkflowFallback(func() error {
				return helper.UpdateClassStatus(*classData, helper.DB_UUID_STATUS_ACTIVE, tenantUUID, schoolUUID)
			}); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Update rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		instanceUUID, err := helper.CreateApproval(*workflow, schoolUUID, tenantUUID, userUUID, &payload.UUID, helper.ACTION_CODE_ACTIVATE, helper.CLASS_ENTITY_TYPE, helper.CLASS_MODULE_CODE, classData)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user approval", nil, err)
		}

		// update status data
		classData.StatusUUID = helper.DB_UUID_STATUS_PENDING
		if err = helper.UpdateClassStatus(*classData, classData.StatusUUID, tenantUUID, schoolUUID); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed make status inactive", nil, err)
		}

		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": payload.UUID, "approval_uuid": instanceUUID}, nil)
	})

	// get
	app.Post(classBaseURL+"/get", func(c fiber.Ctx) error {
		payload := new(model.SearchPayload)
		tenantUUID := c.Get("tenant_uuid")
		//* validate body
		if err := c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}

		if len(tenantUUID) == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or Missing between request body and header", nil, nil)
		}

		data, dataStat, err := helper.SearchClass(tenantUUID, *payload)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}

		result := make(map[string]interface{})

		result["data_statistic"] = dataStat
		result["result"] = data

		return helper.ReturnResponse(c, fiber.StatusOK, "success", result, nil)
	})
}
