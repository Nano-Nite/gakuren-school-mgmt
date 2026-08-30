package v1

import (
	"encoding/json"
	"errors"
	"time"

	"gakuren-system.com/pkg/helper"
	"gakuren-system.com/pkg/model"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func SetupClassRoute(app *fiber.App, API_VERSION string) {
	app.Post(API_VERSION+"/school/class/create", func(c fiber.Ctx) error {
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

		//* get Active Status
		status, err := helper.GetStatusByName(helper.STATUS_ACTIVE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, nil)
		}

		var insertData model.ClassModel
		insertData.UUID = nil
		insertData.Name = payload.Name
		if payload.AbbrName != nil && len(*payload.AbbrName) > 0 {
			insertData.AbbrName = payload.AbbrName
		}
		insertData.Level = payload.Level
		insertData.HomeroomTeacher = payload.HomeroomTeacher
		insertData.StatusUUID = status.UUID
		insertData.CreatedDate = time.Now()
		insertData.UpdatedDate = nil
		insertData.TenantUUID = tenantUUID

		if canBypass { // if user can bypass approval
			if err = helper.InsertClass(insertData); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
			}
		} else { // if user cannot bypass approval
			//* workflow approval logic
			selectedWorkflow, err := helper.DetermineWorkflowApproval(tenantUUID.String(), userUUID.String(), helper.CREATE_CLASS_PERMISSION, helper.ACTION_CODE_CREATE, status.UUID.String())
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
				instance.EntityType = helper.CLASS_ENTITY_TYPE
				instance.EntityUUID = nil
				instance.ActionCode = helper.ACTION_CODE_CREATE
				instance.RequestData = json.RawMessage(payloadJson)
				instance.StatusUUID = status.UUID
				instance.RequestedBy = *userUUID
				instance.FinalizedBy = nil
				instance.FinalizedDate = nil
				instance.UpdatedDate = nil

				// insert approval instance
				instanceUUID, err := helper.CreateApprovalInstance(instance)
				if err != nil {
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
				}

				// ini approval action
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

	app.Post(API_VERSION+"/school/class/get", func(c fiber.Ctx) error {
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
