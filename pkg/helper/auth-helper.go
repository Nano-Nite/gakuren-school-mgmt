package helper

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
	"github.com/golang-jwt/jwt/v5"
)

func ParsePublicKey(base64Key string) (*rsa.PublicKey, error) {
	key := strings.TrimSpace(base64Key)
	if key == "" {
		return nil, fmt.Errorf("public key is empty")
	}

	pemBytes := []byte(key)
	if !strings.Contains(key, "BEGIN") {
		decoded, err := DecodeB64Bytes(key)
		if err == nil {
			pemBytes = decoded
		}
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not RSA public key")
	}

	return rsaPublicKey, nil
}

func ValidateAccessToken(tokenString string) (*model.AccessTokenClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("token is required")
	}

	publicKey, err := ParsePublicKey(os.Getenv("RSA_PUBLIC_KEY"))
	if err != nil {
		return nil, fmt.Errorf("parse RSA public key: %w", err)
	}

	claims := new(model.AccessTokenClaims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Issuer != "" && claims.Issuer != os.Getenv("JWT_ISSUER") {
		return nil, errors.New("invalid issuer")
	}
	if len(claims.Audience) > 0 && claims.Audience[0] != "" && claims.Audience[0] != os.Getenv("JWT_AUDIENCE") {
		return nil, errors.New("invalid audience")
	}

	return claims, nil
}

func GetUserPermission(userUUID string, permission string) (bool, error) {
	query := `
	select distinct p.code from user_sch."user" u 
	join user_sch.role r on u.role_uuid = r.uuid
	join user_sch.role_permission rp on r.uuid = rp.role_uuid
	join user_sch.permission p on rp.permission_uuid = p.uuid
	where u.uuid = $1 and p.code = $2
	limit 1`

	selectedPermission, err := db.GetSingleDataByQuery[model.GetPermission](query, userUUID, permission)
	if err != nil {
		return false, err
	}

	if selectedPermission == nil {
		return false, nil
	}

	return true, nil
}

func ApprovalBypass(userUUID string) (bool, error) {
	query := `
	select distinct p.code from user_sch."user" u 
	join user_sch.role r on u.role_uuid = r.uuid
	join user_sch.role_permission rp on r.uuid = rp.role_uuid
	join user_sch.permission p on rp.permission_uuid = p.uuid
	where u.uuid = $1 and p.code = $2
	limit 1`

	selectedPermission, err := db.GetSingleDataByQuery[model.GetPermission](query, userUUID, APPROVAL_BYPASS)
	if err != nil {
		return false, err
	}

	if selectedPermission == nil {
		return false, nil
	}

	return true, nil
}

func GetUser(uuid string) (*model.UserModel, error) {
	query := `select * from user_sch.user where uuid = $1`
	selectedUser, err := db.GetSingleDataByQuery[model.UserModel](query, uuid)
	if err != nil {
		return nil, err
	}
	return selectedUser, nil
}

func GetRole(uuid string) (*model.RoleModel, error) {
	query := `select * from user_sch.role where uuid = $1`
	selectedUser, err := db.GetSingleDataByQuery[model.RoleModel](query, uuid)
	if err != nil {
		return nil, err
	}
	return selectedUser, nil
}
