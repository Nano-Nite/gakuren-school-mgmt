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

func SetupStudentRoute(app *fiber.App, apiVersion string) {
	studenBaseURL := apiVersion + "/school/student"

	app.Post(studenBaseURL+"/create", func(c fiber.Ctx) error {
		payload := new(model.CreateStudentModel)
		tenantUUID, requesterUUID, err := validateStudentRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}
		if err = c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body or missing role_uuid", nil, err)
		}

		if ok, permissionErr := helper.GetUserPermission(requesterUUID.String(), helper.CREATE_STUDENT_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		// check user existing using email
		if err = helper.UserStudentValidity(*payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Fail to check user", nil, err)
		}

		// get Siswa Role
		studentRole, err := helper.GetRoleByAbbrName(helper.ROLE_STUDENT)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to get student role", nil, err)
		}

		data := model.UserModel{TenantUUID: tenantUUID, Name: payload.Name, Email: payload.Email,
			Phone: payload.Phone, Address: payload.Address, ImgLocation: payload.ImgLocation,
			StatusUUID: helper.DB_UUID_STATUS_ACTIVE, RoleUUID: studentRole.UUID,
			CreatedDate: time.Now()}
		canBypass, err := approvalBypass(requesterUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to check approval bypass", nil, err)
		}
		if canBypass {
			id, insertErr := helper.InsertUserStudent(data)
			if insertErr != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user", nil, insertErr)
			}
			_, err = helper.InsertStudent(*payload, *id, helper.DB_UUID_STATUS_ACTIVE)
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create student", nil, insertErr)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": id}, nil)
		}
		workflow, err := determineStudentWorkflow(tenantUUID, requesterUUID, helper.CREATE_STUDENT_PERMISSION, helper.ACTION_CODE_CREATE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, err)
		}
		if workflow == nil {
			// create user data first then student
			userUUID, insertErr := helper.InsertUserStudent(data)
			if insertErr != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user", nil, insertErr)
			}

			var id *uuid.UUID
			err = executeUserWorkflowFallback(func() error {
				id, err = helper.InsertStudent(*payload, *userUUID, helper.DB_UUID_STATUS_ACTIVE)
				return err
			})
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Create rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": id}, nil)
		}
		approvalUUID, err := createStudentApproval(*workflow, tenantUUID, requesterUUID, nil, helper.ACTION_CODE_CREATE, payload)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user approval", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"approval_uuid": approvalUUID}, nil)
	})

	app.Patch(studenBaseURL+"/update", func(c fiber.Ctx) error {
		payload := new(model.UpdateStudentModel)

		// validation header
		tenantUUID, requesterUUID, err := validateStudentRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}

		// body check
		if err = c.Bind().Body(payload); err != nil || payload.UUID == uuid.Nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body, uuid is required", nil, err)
		}

		// permission check
		if ok, permissionErr := helper.GetUserPermission(requesterUUID.String(), helper.UPDATE_STUDENT_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		// get data
		selectedData, err := helper.GetStudent(payload.UUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Student not found", nil, err)
		}

		// pending data check
		if selectedData.StatusUUID != helper.DB_UUID_STATUS_ACTIVE && selectedData.StatusUUID != helper.DB_UUID_STATUS_INACTIVE {
			return helper.ReturnResponse(c, fiber.StatusConflict, "Update rejected because user has a pending action", nil, nil)
		}

		// activate case
		activate := selectedData.StatusUUID == helper.DB_UUID_STATUS_INACTIVE && strings.EqualFold(payload.Status.String(), helper.DB_UUID_STATUS_ACTIVE.String())
		if activate {
			selectedData.StatusUUID = helper.DB_UUID_STATUS_ACTIVE
		} else {
			selectedData.UUID = payload.UUID
			selectedData.Name = payload.Name
			selectedData.NIS = payload.NIS
			selectedData.NISN = payload.NISN
			selectedData.Phone = payload.Phone
			selectedData.Email = payload.Email
			selectedData.ClassUUID = payload.ClassUUID
			selectedData.Address = payload.Address
			selectedData.GenderUUID = payload.GenderUUID
			selectedData.ParentName = payload.ParentName
			selectedData.ParentEmail = payload.ParentEmail
			selectedData.ParentPhone = payload.ParentPhone
			selectedData.ParentAddress = payload.ParentAddress
		}

		// bypass permission check
		canBypass, err := approvalBypass(requesterUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to check approval bypass", nil, err)
		}
		operation := func() error {
			if activate {
				return helper.UpdateStudentStatus(*selectedData, tenantUUID, payload.Status)
			}
			return helper.UpdateStudent(*selectedData)
		}
		if canBypass {
			if err = operation(); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to update user", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		// workflow check
		workflow, err := determineStudentWorkflow(tenantUUID, requesterUUID, helper.UPDATE_STUDENT_PERMISSION, helper.ACTION_CODE_UPDATE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, err)
		}
		if workflow == nil {
			if err = executeUserWorkflowFallback(operation); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Update rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		// create approval
		approvalUUID, err := createStudentApproval(*workflow, tenantUUID, requesterUUID, &payload.UUID, helper.ACTION_CODE_UPDATE, selectedData)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user update approval", nil, err)
		}

		// update studen status to Pending
		if err = helper.UpdateStudentStatus(*selectedData, tenantUUID, helper.DB_UUID_STATUS_PENDING); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to mark user pending", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": payload.UUID, "approval_uuid": approvalUUID}, nil)
	})

	app.Delete(studenBaseURL+"/delete", func(c fiber.Ctx) error {
		payload := new(model.DeleteStudentModel)

		// validation header
		tenantUUID, requesterUUID, err := validateStudentRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}

		// body check
		if err = c.Bind().Body(payload); err != nil || payload.UUID == uuid.Nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body", nil, err)
		}

		// permission check
		if ok, permissionErr := helper.GetUserPermission(requesterUUID.String(), helper.DELETE_STUDENT_PERMISSION); permissionErr != nil || !ok {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Access Denied", nil, permissionErr)
		}

		// get data
		selectedData, err := helper.GetStudent(payload.UUID, tenantUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusNotFound, "Student not found", nil, err)
		}

		// status validation
		if selectedData.StatusUUID != helper.DB_UUID_STATUS_ACTIVE {
			return helper.ReturnResponse(c, fiber.StatusConflict, "Delete rejected because user is not active", nil, nil)
		}

		// bypass check
		canBypass, err := approvalBypass(requesterUUID)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to check approval bypass", nil, err)
		}
		operation := func() error { return helper.SoftDeleteStudent(*selectedData, tenantUUID) }

		if canBypass {
			if err = operation(); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to delete user", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		workflow, err := determineStudentWorkflow(tenantUUID, requesterUUID, helper.DELETE_STUDENT_PERMISSION, helper.ACTION_CODE_DELETE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, err)
		}
		if workflow == nil {
			if err = executeUserWorkflowFallback(operation); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Delete rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		approvalUUID, err := createStudentApproval(*workflow, tenantUUID, requesterUUID, &payload.UUID, helper.ACTION_CODE_DELETE, selectedData)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user delete approval", nil, err)
		}

		// update status to Pending
		if err = helper.UpdateStudentStatus(*selectedData, tenantUUID, helper.DB_UUID_STATUS_PENDING); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to mark user pending", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": payload.UUID, "approval_uuid": approvalUUID}, nil)
	})

	app.Post(studenBaseURL+"/get", func(c fiber.Ctx) error {
		payload := new(model.SearchPayload)
		tenantUUID, _, err := validateStudentRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}
		if err = c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}
		data, stats, err := helper.SearchStudent(tenantUUID, *payload)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Failed to search users", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"data_statistic": stats, "result": data}, nil)
	})
}

func validateStudentRequest(c fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
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

func approvalBypass(userUUID uuid.UUID) (bool, error) {
	ok, err := helper.ApprovalBypass(userUUID.String())
	if err != nil && err.Error() == "no rows in result set" {
		return false, nil
	}
	return ok, err
}

func determineStudentWorkflow(tenantUUID, userUUID uuid.UUID, permission, action string) (*model.ApprovalWorkflow, error) {
	w, err := helper.DetermineWorkflowApproval(tenantUUID.String(), userUUID.String(), permission, action, helper.DB_UUID_STATUS_ACTIVE.String())
	if err != nil && err.Error() == "no rows in result set" {
		return nil, nil
	}
	return w, err
}

func executeUserWorkflowFallback(operation func() error) error {
	a, err := helper.GetVariableUsingKey(helper.WORKFLOW_NOTFOUND_BEHAVIOUR)
	if err != nil {
		return err
	}
	if a.Value == helper.SAVE {
		return operation()
	}
	if a.Value == helper.REJECT {
		return errors.New("rejected due workflow behaviour")
	}
	return errors.New("skipped due workflow behaviour")
}

func createStudentApproval(workflow model.ApprovalWorkflow, tenantUUID, userUUID uuid.UUID, entityUUID *uuid.UUID, action string, data any) (*uuid.UUID, error) {
	b, err := helper.ConvertModelToJSON(data)
	if err != nil {
		return nil, err
	}
	id, err := helper.CreateApprovalInstance(model.ApprovalInstance{ApprovalWorkflowUUID: workflow.UUID, TenantUUID: tenantUUID, EntityType: helper.STUDENT_ENTITY_TYPE, EntityUUID: entityUUID, ActionCode: action, RequestData: json.RawMessage(b), StatusUUID: helper.DB_UUID_STATUS_ACTIVE, RequestedBy: userUUID})
	if err != nil {
		return nil, err
	}
	_, err = helper.CreateApprovalAction(model.ApprovalAction{ApprovalInstanceUUID: *id, ActionCode: helper.ACTION_CODE_SUBMIT, ActedBy: userUUID, CreatedDate: time.Now()})
	return id, err
}
