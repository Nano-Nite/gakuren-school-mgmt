package auth

/*
var JWTService *model.JWTService

func NewJWTService() error {
	privateKey, err := helper.ParsePrivateKey(os.Getenv("RSA_PRIVATE_KEY"))
	if err != nil {
		return fmt.Errorf("parse JWT private key: %w", err)
	}

	service := new(model.JWTService)
	service.PrivateKey = privateKey

	JWTService = service

	log.Println("JWT Service Started")
	return nil
}

func AuthenticateBearerToken(authHeader string) (string, error) {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("authorization header must be in format: Bearer <token>")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("bearer token is required")
	}

	expectedToken := strings.TrimSpace(os.Getenv("BEARER_TOKEN"))
	if expectedToken == "" {
		expectedToken = "dev-token"
	}

	if token != expectedToken {
		return "", errors.New("invalid bearer token")
	}

	return token, nil
}

func GetPublicKey() (string, error) {
	publicKeyEnv := strings.TrimSpace(os.Getenv("RSA_PUBLIC_KEY"))
	if publicKeyEnv == "" {
		return "", errors.New("public key is not set in environment variables")
	}

	publicKeyPem, err := decodePEM(publicKeyEnv)
	if err != nil {
		publicKeyPem = publicKeyEnv
	}

	publicKeyPem = strings.TrimSpace(publicKeyPem)
	publicKeyPem = strings.ReplaceAll(publicKeyPem, `\n`, "\n")

	if !strings.Contains(publicKeyPem, "-----BEGIN PUBLIC KEY-----") && !strings.Contains(publicKeyPem, "-----BEGIN RSA PUBLIC KEY-----") {
		return "", errors.New("RSA_PUBLIC_KEY does not contain a valid PEM public key")
	}

	return publicKeyPem, nil
}

func decodePEM(src string) (string, error) {
	tmp := strings.TrimSpace(src)
	if tmp == "" {
		return "", errors.New("empty base64 input")
	}

	decoded, err := helper.DecodeB64Bytes(tmp)
	if err == nil {
		return string(decoded), nil
	}

	decoded, err = base64.URLEncoding.DecodeString(tmp)
	if err == nil {
		return string(decoded), nil
	}

	decoded, err = base64.RawStdEncoding.DecodeString(tmp)
	if err == nil {
		return string(decoded), nil
	}

	decoded, err = base64.RawURLEncoding.DecodeString(tmp)
	if err == nil {
		return string(decoded), nil
	}

	return "", err
}

func GenerateAccessToken(userID string, username string, roles []string, s model.JWTService) (*jwt.Token, error) {
	now := time.Now()
	duration, err := strconv.Atoi(os.Getenv("ACCESS_TOKEN_DURATION"))
	if err != nil {
		return nil, fmt.Errorf("sign JWT: %w", err)
	}

	JWTIssuer := os.Getenv("JWT_ISSUER")
	JWTAudience := os.Getenv("JWT_AUDIENCE")
	JWTKeyID := os.Getenv("JWT_KEY_ID")
	AccessTokenDuration := time.Duration(duration) * time.Minute

	claims := model.AccessTokenClaims{
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    JWTIssuer,
			Audience:  jwt.ClaimStrings{JWTAudience},
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = JWTKeyID

	signed, err := token.SignedString(s.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("sign JWT: %w", err)
	}
	token.Raw = signed

	return token, nil
}

func GenerateRefreshToken(userID string, username string) (*jwt.Token, error) {
	JWTIssuer := os.Getenv("JWT_ISSUER")
	JWTAudience := os.Getenv("JWT_AUDIENCE")
	JWTKeyID := os.Getenv("JWT_KEY_ID")

	refreshTokenDuration, _ := strconv.Atoi(os.Getenv("REFRESH_TOKEN_DURATION"))
	refreshTokenExp := time.Now().Add(time.Duration(refreshTokenDuration) * 24 * time.Hour)

	refreshClaims := model.RefreshClaims{
		Username: username,
		TokenID:  uuid.New().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    JWTIssuer,
			Audience:  jwt.ClaimStrings{JWTAudience},
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(refreshTokenExp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshKey, _ := helper.GenerateSecureKey()

	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenObj.Header["kid"] = JWTKeyID

	refreshTokenStr, err := refreshTokenObj.SignedString([]byte(refreshKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}
	refreshTokenObj.Raw = refreshTokenStr

	return refreshTokenObj, nil
}

func UpsertRefreshToken(userID string, accessToken *jwt.Token, refreshToken *jwt.Token, c fiber.Ctx) error {
	tx, err := db.Conn.Begin(db.DBCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	query := `SELECT * FROM public.refresh_session WHERE user_uuid = $1 and revoke_date is null order by created_date DESC LIMIT 1`
	selectedRefreshData, err := db.GetSingleDataByQuery[model.RefreshTokenModel](query, userID)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return err
		}
	}

	query = `INSERT INTO public.refresh_session (user_uuid, token_hash, expired_date, user_agent, ip_address, access_token_hash, access_token_expired_date) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING uuid`
	accessExpDate, _ := accessToken.Claims.GetExpirationTime()
	refreshExpDate, _ := refreshToken.Claims.GetExpirationTime()
	returnUUID, err := db.InsertReturnUUID(query, userID, refreshToken.Raw, refreshExpDate.Format(time.RFC3339), c.UserAgent(), c.IP(), accessToken.Raw, accessExpDate.Format(time.RFC3339))
	if err != nil {
		return err
	}

	if selectedRefreshData != nil {
		query = `UPDATE public.refresh_session SET revoke_date=$1, replaced_by=$2 WHERE uuid=$3`
		if err = db.ExecuteQuery(query, time.Now().Format(time.RFC3339), returnUUID, selectedRefreshData.UUID); err != nil {
			return err
		}
	}

	return tx.Commit(db.DBCtx)
}

func GetLoginDataByEmail(email string) (*model.LoginData, error) {
	loginData, err := db.GetSingleDataByQuery[model.LoginData](`
	select
		t.code,
		t.name tenant_name,
		t.timezone,
		t.version tenant_version,
		u.name user_name,
		u.email,
		u.phone,
		u.address,
		u.version user_version,
		r.name role_name,
		r.level role_level
	from user_sch.tenant t
	join user_sch.user u on t.uuid = u.tenant_uuid
	join public.role r on u.role_uuid = r.uuid
	where u.email = $1
	`, email)
	if err != nil {
		return nil, err
	}
	return loginData, nil
}

func UpdateRefreshTokenAccessToken(userID string, accessToken *jwt.Token, c fiber.Ctx) error {
	tx, err := db.Conn.Begin(db.DBCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	query := `SELECT * FROM public.refresh_session WHERE user_uuid = $1 and revoke_date is null order by created_date DESC LIMIT 1`
	selectedRefreshData, err := db.GetSingleDataByQuery[model.RefreshTokenModel](query, userID)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return err
		}
	}

	if selectedRefreshData != nil {
		query = `UPDATE public.refresh_session SET access_token_hash=$1, access_token_expired_date=$2 WHERE uuid=$3`
		if err = db.ExecuteQuery(query, accessToken.Raw, time.Now().Format(time.RFC3339), selectedRefreshData.UUID); err != nil {
			return err
		}
	}

	return tx.Commit(db.DBCtx)
}
*/
