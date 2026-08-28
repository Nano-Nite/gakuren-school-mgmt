package v1

import (
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
		//* validate body
		if err := c.Bind().Body(payload); err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
		}

		if len(payload.Name) == 0 || payload.Level == 0 || len(tenantUUIDHeader) == 0 {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or Missing between request body and header", nil, nil)
		}

		tenantUUID, err := uuid.Parse(tenantUUIDHeader)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or Missing between request body and header", nil, err)
		}

		status, err := helper.GetStatusByName(helper.STATUS_ACTIVE)
		if err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, nil)
		}

		var insertData model.ClassModel
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

		if err = helper.InsertClass(insertData); err != nil {
			return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, try again in a while", nil, err)
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
