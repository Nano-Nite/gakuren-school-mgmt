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
		tenantUUIDHeader := c.Get("tenant_uuid")
		authHeader := c.Get("Authorization")

		//* validate user using access token
		userUUID, err := helper.GetUserUUIDByAccessToken(authHeader)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, err)
		}
		if userUUID == nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, nil)
		}

		//* validate tenant id
		tenantUUID, err := uuid.Parse(tenantUUIDHeader)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or Missing between request body and header", nil, err)
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
		if len(payload.Name) == 0 || payload.Level == 0 || len(tenantUUIDHeader) == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or Missing between request body and header", nil, nil)
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

		if canBypass { // if user can bypass approval
			if err = helper.InsertClass(insertData); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
			}
		} else { // if user cannot bypass approval
			//* workflow approval logic
			selectedWorkflow, err := helper.DetermineWorkflowApproval(tenantUUID.String(), userUUID.String(), helper.CREATE_CLASS_PERMISSION, helper.ACTION_CODE_CREATE, helper.DB_UUID_STATUS_ACTIVE.String())
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
					if err = helper.InsertClass(insertData); err != nil {
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
				instance.TenantUUID = tenantUUID
				instance.EntityType = helper.CLASS_ENTITY_TYPE
				instance.EntityUUID = nil
				instance.ActionCode = helper.ACTION_CODE_CREATE
				instance.RequestData = json.RawMessage(payloadJson)
				instance.StatusUUID = helper.DB_UUID_STATUS_ACTIVE
				instance.RequestedBy = *userUUID
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
				action.ActedBy = *userUUID
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
		tenantUUID, userUUID, err := validateClassRequest(c)
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
		classData, err := helper.GetClass(payload.UUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Class not found", nil, err)
		}

		// status check
		if classData.StatusUUID != helper.DB_UUID_STATUS_ACTIVE && classData.StatusUUID != helper.DB_UUID_STATUS_INACTIVE {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Update rejected due current status is pending", nil, err)
		}

		if classData.StatusUUID == helper.DB_UUID_STATUS_INACTIVE && strings.EqualFold(payload.Status, helper.STATUS_ACTIVE) { // activate case
			classData.StatusUUID = helper.DB_UUID_STATUS_ACTIVE
		} else {
			classData.Name = payload.Name
			classData.AbbrName = payload.AbbrName
			classData.Level = payload.Level
			classData.HomeroomTeacher = payload.HomeroomTeacher
		}

		// bypass check
		canBypass, bypassErr := helper.ApprovalBypass(userUUID.String())
		if bypassErr != nil && bypassErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, bypassErr)
		}
		if canBypass {
			if err = helper.UpdateClass(*classData); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to update class", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		workflow, workflowErr := helper.DetermineWorkflowApproval(tenantUUID.String(), userUUID.String(), helper.UPDATE_CLASS_PERMISSION, helper.ACTION_CODE_UPDATE, helper.DB_UUID_STATUS_ACTIVE.String())
		if workflowErr != nil && workflowErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, workflowErr)
		}
		if workflow == nil {
			if err = executeClassWorkflowFallback(func() error { return helper.UpdateClass(*classData) }); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Update rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		instanceUUID, err := createClassApproval(*workflow, tenantUUID, userUUID, &payload.UUID, helper.ACTION_CODE_UPDATE, classData, helper.DB_UUID_STATUS_ACTIVE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create class update approval", nil, err)
		}

		// update status data
		classData.StatusUUID = helper.DB_UUID_STATUS_PENDING
		if err = helper.UpdateClassStatus(*classData); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed make status inactive", nil, err)
		}

		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": payload.UUID, "approval_uuid": instanceUUID}, nil)
	})

	// delete
	app.Delete(classBaseURL+"/delete", func(c fiber.Ctx) error {
		payload := new(model.DeleteClassModel)
		tenantUUID, userUUID, err := validateClassRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}
		if err = c.Bind().Body(payload); err != nil || payload.UUID == uuid.Nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}
		if ok, permissionErr := helper.GetUserPermission(userUUID.String(), helper.DELETE_CLASS_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		classData, err := helper.GetClass(payload.UUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Class not found", nil, err)
		}

		// status check
		if classData.StatusUUID != helper.DB_UUID_STATUS_ACTIVE {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Delete rejected due current status is not active", nil, err)
		}

		canBypass, bypassErr := helper.ApprovalBypass(userUUID.String())
		if bypassErr != nil && bypassErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, bypassErr)
		}
		if canBypass {
			if err = helper.SoftDeleteClass(payload.UUID, tenantUUID); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to delete class", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		workflow, workflowErr := helper.DetermineWorkflowApproval(tenantUUID.String(), userUUID.String(), helper.DELETE_CLASS_PERMISSION, helper.ACTION_CODE_DELETE, helper.DB_UUID_STATUS_ACTIVE.String())
		if workflowErr != nil && workflowErr.Error() != "no rows in result set" {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, workflowErr)
		}
		if workflow == nil {
			if err = executeClassWorkflowFallback(func() error { return helper.SoftDeleteClass(payload.UUID, tenantUUID) }); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Delete rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		instanceUUID, err := createClassApproval(*workflow, tenantUUID, userUUID, &payload.UUID, helper.ACTION_CODE_DELETE, classData, helper.DB_UUID_STATUS_ACTIVE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create class delete approval", nil, err)
		}

		// update status data
		classData.StatusUUID = helper.DB_UUID_STATUS_PENDING
		if err = helper.UpdateClassStatus(*classData); err != nil {
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

func validateClassRequest(c fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	tenantUUID, err := uuid.Parse(c.Get("tenant_uuid"))
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("invalid or missing tenant UUID")
	}
	userUUID, err := helper.GetUserUUIDByAccessToken(c.Get("Authorization"))
	if err != nil || userUUID == nil {
		if err == nil {
			err = errors.New("missing user UUID")
		}
		return uuid.Nil, uuid.Nil, err
	}
	return tenantUUID, *userUUID, nil
}

func executeClassWorkflowFallback(operation func() error) error {
	action, err := helper.GetVariableUsingKey(helper.WORKFLOW_NOTFOUND_BEHAVIOUR)
	if err != nil {
		return err
	}
	switch action.Value {
	case helper.SAVE:
		return operation()
	case helper.REJECT:
		return errors.New("rejected due workflow behaviour")
	default:
		return errors.New("skipped due workflow behaviour")
	}
}

func createClassApproval(
	workflow model.ApprovalWorkflow,
	tenantUUID, userUUID uuid.UUID,
	entityUUID *uuid.UUID,
	actionCode string,
	requestData any,
	statusUUID uuid.UUID,
) (*uuid.UUID, error) {
	payloadJSON, err := helper.ConvertModelToJSON(requestData)
	if err != nil {
		return nil, err
	}

	instanceUUID, err := helper.CreateApprovalInstance(model.ApprovalInstance{
		ApprovalWorkflowUUID: workflow.UUID,
		TenantUUID:           tenantUUID,
		EntityType:           helper.CLASS_ENTITY_TYPE,
		EntityUUID:           entityUUID,
		ActionCode:           actionCode,
		RequestData:          json.RawMessage(payloadJSON),
		StatusUUID:           statusUUID,
		RequestedBy:          userUUID,
	}, helper.CLASS_MODULE_CODE)
	if err != nil {
		return nil, err
	}

	_, err = helper.CreateApprovalAction(model.ApprovalAction{
		ApprovalInstanceUUID: *instanceUUID,
		ActionCode:           helper.ACTION_CODE_SUBMIT,
		ActedBy:              userUUID,
		CreatedDate:          time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return instanceUUID, nil
}
