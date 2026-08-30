package v1

import (
	"gakuren-system.com/pkg/helper"
	"gakuren-system.com/pkg/model"
	"github.com/gofiber/fiber/v3"
)

func SetupApprovalRoute(app *fiber.App, API_VERSION string) {
	app.Post(API_VERSION+"/school/approval/my-approval", func(c fiber.Ctx) error {
		payload := new(model.SearchPayload)
		// tenantUUID := c.Get("tenant_uuid")
		authHeader := c.Get("Authorization")

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
}
