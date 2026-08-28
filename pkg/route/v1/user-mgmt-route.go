package v1

import (
	"github.com/gofiber/fiber/v3"
)

func SetupUserManagementRoutes(app *fiber.App, API_VERSION string) {
	/*
		app.Post(API_VERSION+"/auth/login", func(c fiber.Ctx) error {
			authHeader := c.Get("Authorization")
			payload := new(model.LoginPayload)

			//* validate header
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, nil)
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			//* validate body
			if err := c.Bind().Body(payload); err != nil {
				return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
			}

			//* decode RSA
			pwd, err := helper.DecodeRSA(tokenString)
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or malformed token", nil, err)
			}

			//* get user by email
			selectedUser, err := db.GetSingleDataByQuery[model.UserModel]("select * from user_sch.user where email = $1", payload.Email)
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Invalid username or password", nil, nil)
			}

			//* get user login by username
			selectedUserLogin, err := db.GetSingleDataByQuery[model.UserLoginModel]("select * from user_sch.user_login where username = $1", payload.Email)
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Invalid username or password", nil, nil)
			}

			// logged in status
			// loginStatus, err := helper.GetStatusByName("logged in")
			// if err != nil {
			// 	return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, please try again later", nil, err)
			// }
			// if selectedUserLogin.StatusUUID == loginStatus.UUID {
			// 	return helper.ReturnResponse(c, fiber.StatusUnauthorized, "User already logged in", nil, nil)
			// }

			userPassword := selectedUser.Email + "|" + selectedUser.TenantUUID.String() + "|" + string(pwd)

			valid, err := auth.VerifyPassword(userPassword, selectedUserLogin.Password)
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Invalid username or password", nil, err)
			}

			if !valid {
				return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Invalid username or password", nil, nil)
			}

			selectedStatus, err := helper.GetStatusByName("logged out")
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, please try again later", nil, err)
			}

			result := make(map[string]interface{})
			token := make(map[string]interface{})

			accessTokenExpired, _ := strconv.Atoi(os.Getenv("ACCESS_TOKEN_DURATION"))
			refreshTokenExpired, _ := strconv.Atoi(os.Getenv("REFRESH_TOKEN_DURATION"))

			loginData, err := auth.GetLoginDataByEmail(selectedUser.Email)
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, please try again later", nil, err)
			}

			// Set user permission
			selectedPermission, err := helper.GetUserRolePermissionCodeList(selectedUser.UUID.String())
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, please try again later", nil, err)
			}

			// Set user menu
			selectedMenu, err := helper.GetUserMenuList(selectedUser.UUID.String())
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Internal server error, please try again later", nil, err)
			}

			// logged out status
			if selectedUserLogin.StatusUUID == selectedStatus.UUID {
				//* generate new access token and refresh token
				accessToken, err := auth.GenerateAccessToken(selectedUser.UUID.String(), selectedUser.Email, []string{"admin"}, *auth.JWTService)
				if err != nil || accessToken == nil {
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to generate access token", nil, err)
				}

				refreshToken, err := auth.GenerateRefreshToken(selectedUser.UUID.String(), selectedUser.Email)
				if err != nil || refreshToken == nil {
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to generate refresh token", nil, err)
				}

				if err = auth.UpsertRefreshToken(selectedUser.UUID.String(), accessToken, refreshToken, c); err != nil {
					return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to store refresh token", nil, err)
				}

				//* update user_login data
				if err = db.ExecuteQuery("UPDATE user_sch.user_login SET last_login = now(), updated_date = now(), failed_attempt = 0, status_uuid=(SELECT uuid FROM public.status s WHERE LOWER(s.name) = 'logged in') WHERE uuid = $1", selectedUserLogin.UUID.String()); err != nil {
					return helper.ReturnResponse(c, fiber.StatusBadRequest, "User not found", nil, nil)
				}

				token["access_token"] = accessToken.Raw
				token["token_type"] = "bearer"
				token["expired_in"] = accessTokenExpired * 60
				token["refresh_token"] = refreshToken.Raw
				token["refresh_expired_in"] = 3600 * 24 * refreshTokenExpired

				result["user_data"] = loginData
				result["token"] = token
				result["permission"] = selectedPermission
				result["menu"] = selectedMenu

				return helper.ReturnResponse(c, fiber.StatusOK, "Login successful", result, nil)
			} else {
				//* get refresh_session data
				selectedData, err := db.GetSingleDataByQuery[model.RefreshTokenModel]("SELECT * FROM public.refresh_session WHERE user_uuid = $1 and revoke_date is null order by created_date DESC LIMIT 1", selectedUser.UUID.String())
				if err != nil {
					return helper.ReturnResponse(c, fiber.StatusBadRequest, "User not found", nil, nil)
				}

				if time.Now().Compare(*selectedData.AccessTokenExpiredDate) < 0 { // active access token
					//* generate new access token and keep refresh token
					// accessToken, err := auth.GenerateAccessToken(selectedUser.UUID.String(), selectedUser.Email, []string{"admin"}, *auth.JWTService)
					// if err != nil || accessToken == nil {
					// 	return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to generate access token", nil, err)
					// }

					token["access_token"] = string(selectedData.AccessTokenHash)
					token["expired_in"] = int64(time.Until(*selectedData.AccessTokenExpiredDate)) * 60
					token["refresh_expired_in"] = int64(time.Until(*selectedData.ExpiredDate).Seconds())
					token["token_type"] = "bearer"
					token["refresh_token"] = string(selectedData.TokenHash)

					result["user_data"] = loginData
					result["token"] = token
					result["permission"] = selectedPermission
					result["menu"] = selectedMenu

					// if err = auth.UpdateRefreshTokenAccessToken(selectedUser.UUID.String(), accessToken, c); err != nil {
					// 	return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to store access token", nil, err)
					// }
					return helper.ReturnResponse(c, fiber.StatusOK, "Login successful", result, nil)
				} else { // inactive access token
					if time.Now().Compare(*selectedData.ExpiredDate) < 0 { // refresh token active
						//* generate new access token and keep refresh token
						accessToken, err := auth.GenerateAccessToken(selectedUser.UUID.String(), selectedUser.Email, []string{"admin"}, *auth.JWTService)
						if err != nil || accessToken == nil {
							return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to generate access token", nil, err)
						}

						token["access_token"] = accessToken.Raw
						token["expired_in"] = accessTokenExpired * 60
						token["refresh_expired_in"] = int64(time.Until(*selectedData.ExpiredDate).Seconds())
						token["token_type"] = "bearer"
						token["refresh_token"] = selectedData.TokenHash

						result["user_data"] = loginData
						result["token"] = token
						result["permission"] = selectedPermission
						result["menu"] = selectedMenu

						if err = auth.UpdateRefreshTokenAccessToken(selectedUser.UUID.String(), accessToken, c); err != nil {
							return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to store access token", nil, err)
						}

						return helper.ReturnResponse(c, fiber.StatusOK, "Login successful", result, nil)
					} else { //refresh token inactive
						//* generate access token and refresh token
						accessToken, err := auth.GenerateAccessToken(selectedUser.UUID.String(), selectedUser.Email, []string{"admin"}, *auth.JWTService)
						if err != nil || accessToken == nil {
							return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to generate access token", nil, err)
						}

						refreshToken, err := auth.GenerateRefreshToken(selectedUser.UUID.String(), selectedUser.Email)
						if err != nil || refreshToken == nil {
							return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to generate refresh token", nil, err)
						}

						if err = auth.UpsertRefreshToken(selectedUser.UUID.String(), accessToken, refreshToken, c); err != nil {
							return helper.ReturnResponse(c, fiber.StatusInternalServerError, "Failed to store refresh token", nil, err)
						}

						//* update user_login data
						if err = db.ExecuteQuery("UPDATE user_sch.user_login SET last_login = now(), updated_date = now(), failed_attempt = 0, status_uuid=(SELECT uuid FROM public.status s WHERE LOWER(s.name) = 'logged in') WHERE uuid = $1", selectedUserLogin.UUID.String()); err != nil {
							return helper.ReturnResponse(c, fiber.StatusBadRequest, "User not found", nil, nil)
						}

						token["access_token"] = accessToken.Raw
						token["token_type"] = "bearer"
						token["expired_in"] = accessTokenExpired * 60
						token["refresh_token"] = refreshToken.Raw
						token["refresh_expired_in"] = 3600 * 24 * refreshTokenExpired

						result["user_data"] = loginData
						result["token"] = token
						result["permission"] = selectedPermission
						result["menu"] = selectedMenu

						return helper.ReturnResponse(c, fiber.StatusOK, "Login successful", result, nil)
					}
				}
			}
		})

		app.Post(API_VERSION+"/auth/logout", func(c fiber.Ctx) error {
			authHeader := c.Get("Authorization")
			payload := new(model.LogoutPayload)

			//* validate header
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return helper.ReturnResponse(c, fiber.StatusUnauthorized, "Missing or invalid token", nil, nil)
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			//* validate body
			if err := c.Bind().Body(payload); err != nil {
				return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid request body format", nil, err)
			}

			//* validate token
			_, err := auth.ValidateAccessToken(tokenString)
			if err != nil {
				if !strings.Contains(err.Error(), "is expired") {
					return helper.ReturnResponse(c, fiber.StatusBadRequest, "Invalid or malformed token", nil, err)
				}
			}

			//* get user by email
			selectedUser, err := db.GetSingleDataByQuery[model.UserModel]("select * from user_sch.user where email = $1", payload.Email)
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusBadRequest, "User not found", nil, nil)
			}

			//* get user_login by username
			selectedUserLogin, err := db.GetSingleDataByQuery[model.UserLoginModel]("select * from user_sch.user_login where username = $1 and status_uuid = (select uuid from public.status s where lower(name) = 'logged in')", selectedUser.Email)
			if err != nil {
				return helper.ReturnResponse(c, fiber.StatusBadRequest, "User already logged out", nil, nil)
			}

			//* update user_login data
			if err = db.ExecuteQuery("UPDATE user_sch.user_login SET last_logout = now(), updated_date = now(), status_uuid = (SELECT uuid FROM public.status s WHERE LOWER(s.name) = 'logged out') WHERE uuid = $1", selectedUserLogin.UUID.String()); err != nil {
				return helper.ReturnResponse(c, fiber.StatusBadRequest, "User session not found", nil, nil)
			}

			//* revoke refresh_session data
			if err = db.ExecuteQuery("UPDATE public.refresh_session SET revoke_date = NOW() WHERE uuid = (SELECT uuid FROM public.refresh_session WHERE user_uuid = $1 AND revoke_date IS NULL ORDER BY created_date DESC LIMIT 1)", selectedUserLogin.UserUUID.String()); err != nil {
				return helper.ReturnResponse(c, fiber.StatusBadRequest, "User not found", nil, nil)
			}

			return helper.ReturnResponse(c, fiber.StatusOK, "Logout successful", nil, nil)
		})
	*/
}
