package v1

import (
	"log"
	"os"
	"strings"

	"gakuren-system.com/pkg/helper"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func SetupRoutes() {
	app := fiber.New()

	app.Use(cors.New())

	app.Use(func(c fiber.Ctx) error {
		log.Printf("API hit : %s %s <> IP Address : %s <> User Agent : %s\n", c.Method(), c.OriginalURL(), c.IP(), c.UserAgent())
		log.Println("Authorization : ", c.Get("Authorization"))
		log.Println("Body : ", string(c.Req().Body()))

		// authHeader := c.Get("Authorization")
		//* validate header
		// if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		// 	return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, nil)
		// }
		// tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		//* validate token
		// _, err := helper.ValidateAccessToken(tokenString)
		// if err != nil {
		// 	if strings.Contains(err.Error(), "is expired") {
		// 		return helper.ReturnResponse(c, fiber.StatusForbidden, "Expired access token", nil, err)
		// 	} else {
		// 		log.Println(err.Error())
		// 	}
		// }

		return c.Next()
	})

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("ready to go !!!!!!!!!!")
	})

	// child route setup
	SetupClassRoute(app, helper.API_VERSION)
	SetupApprovalRoute(app, helper.API_VERSION)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
