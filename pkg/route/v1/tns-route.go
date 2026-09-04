package v1

import (
	"strings"
	"time"

	"gakuren-system.com/pkg/helper"
	"gakuren-system.com/pkg/model"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func SetupTNSRoute(app *fiber.App, apiVersion string) {
	studenBaseURL := apiVersion + "/school/tns"

	app.Post(studenBaseURL+"/create", func(c fiber.Ctx) error {
		payload := new(model.CreateStudentModel)
		tenantUUID, requesterUUID, err := helper.ValidateRequest(c)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid authentication data", nil, err)
		}
		if err = c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body", nil, err)
		}

		if ok, permissionErr := helper.GetUserPermission(requesterUUID.String(), helper.CREATE_TNS_PERMISSION); permissionErr != nil || !ok {
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
		canBypass, err := helper.ValidateApprovalBypass(requesterUUID)
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
		workflow, err := helper.DetermineWorkflow(tenantUUID, requesterUUID, helper.CREATE_STUDENT_PERMISSION, helper.ACTION_CODE_CREATE)
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
			err = helper.ExecuteWorkflowFallback(func() error {
				id, err = helper.InsertStudent(*payload, *userUUID, helper.DB_UUID_STATUS_ACTIVE)
				return err
			})
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Create rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"uuid": id}, nil)
		}
		approvalUUID, err := helper.CreateApproval(*workflow, tenantUUID, requesterUUID, nil, helper.ACTION_CODE_CREATE, helper.STUDENT_ENTITY_TYPE, helper.STUDENT_MODULE_CODE, payload)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to create user approval", nil, err)
		}
		return helper.ReturnResponse(c, fiber.StatusOK, "success", map[string]any{"approval_uuid": approvalUUID}, nil)
	})

	app.Patch(studenBaseURL+"/update", func(c fiber.Ctx) error {
		payload := new(model.UpdateStudentModel)

		// validation header
		tenantUUID, requesterUUID, err := helper.ValidateRequest(c)
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
		canBypass, err := helper.ValidateApprovalBypass(requesterUUID)
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
		workflow, err := helper.DetermineWorkflow(tenantUUID, requesterUUID, helper.UPDATE_STUDENT_PERMISSION, helper.ACTION_CODE_UPDATE)
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
		approvalUUID, err := helper.CreateApproval(*workflow, tenantUUID, requesterUUID, &payload.UUID, helper.ACTION_CODE_UPDATE, helper.STUDENT_ENTITY_TYPE, helper.STUDENT_MODULE_CODE, selectedData)
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
		tenantUUID, requesterUUID, err := helper.ValidateRequest(c)
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
		canBypass, err := helper.ValidateApprovalBypass(requesterUUID)
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

		workflow, err := helper.DetermineWorkflow(tenantUUID, requesterUUID, helper.DELETE_STUDENT_PERMISSION, helper.ACTION_CODE_DELETE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to determine approval workflow", nil, err)
		}
		if workflow == nil {
			if err = helper.ExecuteWorkflowFallback(operation); err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Delete rejected by workflow configuration", nil, err)
			}
			return helper.ReturnResponse(c, fiber.StatusOK, "success", payload, nil)
		}

		approvalUUID, err := helper.CreateApproval(*workflow, tenantUUID, requesterUUID, &payload.UUID, helper.ACTION_CODE_DELETE, helper.STUDENT_ENTITY_TYPE, helper.STUDENT_MODULE_CODE, selectedData)
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
		tenantUUID, _, err := helper.ValidateRequest(c)
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
