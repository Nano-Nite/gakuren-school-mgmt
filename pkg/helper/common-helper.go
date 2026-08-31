package helper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"strings"
	"time"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func ReturnResponse(c fiber.Ctx, statusCode int, message string, data interface{}, err error) error {
	response := make(map[string]interface{})
	response["message"] = message
	response["data"] = nil
	response["error"] = nil

	if data != nil {
		response["data"] = data
	}
	if err != nil {
		log.Println(err)
		response["error"] = err.Error()
	}
	return c.Status(statusCode).JSON(response)
}

func DecodeB64String(src string) (string, error) {
	decodedByte, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return "", err
	}

	return string(decodedByte), nil
}

func DecodeB64Bytes(src string) ([]byte, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil, errors.New("empty base64 input")
	}

	b, err := base64.StdEncoding.DecodeString(src)
	if err == nil {
		return b, nil
	}

	b, err = base64.URLEncoding.DecodeString(src)
	if err == nil {
		return b, nil
	}

	return nil, err
}

func CalculateDataStatisticResult(count *model.CountResult, payload model.SearchPayload, totalData int) model.DataStatistics {
	var dataStat model.DataStatistics
	dataStat.TotalRow = count.Count
	if payload.RowPerPage != nil {
		dataStat.RowPerPage = *payload.RowPerPage
	} else {
		dataStat.RowPerPage = DEFAULT_ROW_PER_PAGES
	}
	if payload.Page != nil {
		dataStat.CurrentPage = *payload.Page
	} else {
		dataStat.CurrentPage = DEFAULT_PAGES
	}
	dataStat.MaxPage = int(math.Ceil(float64(count.Count) / float64(dataStat.RowPerPage)))
	dataStat.CurrentRow = totalData
	if (dataStat.CurrentPage * dataStat.RowPerPage) > dataStat.TotalRow {
		dataStat.StartRow = dataStat.TotalRow - dataStat.CurrentRow + 1
	} else {
		dataStat.StartRow = (dataStat.CurrentPage * dataStat.RowPerPage) - (dataStat.RowPerPage - 1)
	}
	if dataStat.CurrentPage*dataStat.RowPerPage < dataStat.TotalRow {
		dataStat.EndRow = dataStat.CurrentPage * dataStat.RowPerPage
	} else {
		dataStat.EndRow = dataStat.TotalRow
	}

	return dataStat
}

func GetUserUUIDByAccessToken(authHeader string) (*uuid.UUID, error) {
	// authHeader := c.Get("Authorization")

	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, errors.New("Missing or invalid token")
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("token is required")
	}

	publicKey, err := ParsePublicKey(os.Getenv("RSA_PUBLIC_KEY"))
	if err != nil {
		return nil, fmt.Errorf("parse RSA public key: %w", err)
	}

	claims := new(model.AccessTokenClaims)
	_, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if claims.RegisteredClaims.Subject == "" {
		return nil, errors.New("missing subject in token claims")
	}

	subjectUUID, err := uuid.Parse(claims.RegisteredClaims.Subject)
	if err != nil {
		return nil, fmt.Errorf("parse subject as UUID: %w", err)
	}
	return &subjectUUID, nil
}

func GetVariableUsingValue(actionCodeValue string) (*model.VariableModel, error) {
	query := `select * from public.variable v where v.value = $1`
	selectedVariable, err := db.GetSingleDataByQuery[model.VariableModel](query, actionCodeValue)
	if err != nil {
		return nil, err
	}
	return selectedVariable, nil
}

func GetVariableUsingKey(key string) (*model.VariableModel, error) {
	query := `select * from public.variable v where v.key = $1`
	selectedVariable, err := db.GetSingleDataByQuery[model.VariableModel](query, key)
	if err != nil {
		return nil, err
	}
	return selectedVariable, nil
}

func GetMenuUUID(userUUID string, permissionCode string) (*model.GetMenuUUID, error) {
	query := `
	select m.uuid from user_sch."user" u 
	join user_sch.role r on u.role_uuid = r.uuid
	join user_sch.role_permission rp on r.uuid = rp.role_uuid
	join user_sch.permission p on rp.permission_uuid = p.uuid
	join user_sch.menu m on p.menu_uuid = m.uuid
	where u.uuid = $1 and p.code = $2
	limit 1
	`
	selectedMenuUUID, err := db.GetSingleDataByQuery[model.GetMenuUUID](query, userUUID, permissionCode)
	if err != nil {
		return nil, err
	}
	return selectedMenuUUID, nil
}

func GetWorkflowApproval(tenantUUID string, actionCode string, menuUUID string, statusUUID string) (*model.ApprovalWorkflow, error) {
	query := `
	select * from approval.approval_workflow aw 
	where aw.tenant_uuid = $1
	and upper(aw.action_code)  = upper($2)
	and aw.menu_uuid = $3
	and aw.status_uuid = $4
	`
	selectedWorkflow, err := db.GetSingleDataByQuery[model.ApprovalWorkflow](query, tenantUUID, actionCode, menuUUID, statusUUID)
	if err != nil {
		return nil, err
	}
	return selectedWorkflow, nil
}

func DetermineWorkflowApproval(tenantUUID string, userUUID string, permissionCode string, actionCode string, status string) (*model.ApprovalWorkflow, error) {
	//* get menu id
	selectedMenuUUID, err := GetMenuUUID(userUUID, permissionCode)
	if err != nil {
		return nil, err
	}

	if selectedMenuUUID == nil {
		return nil, err
	}

	//* get action code
	selectedActionCode, err := GetVariableUsingValue(actionCode)
	if err != nil {
		return nil, err
	}

	if selectedActionCode == nil {
		return nil, err
	}

	//* get workflow
	selectedWorkflow, err := GetWorkflowApproval(tenantUUID, selectedActionCode.Value, selectedMenuUUID.UUID.String(), status)
	if err != nil {
		return nil, err
	}

	if selectedWorkflow == nil {
		return nil, err
	}

	return selectedWorkflow, nil
}

func ConvertModelToJSON(data any) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("convert model to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

func CreateApprovalInstance(instance model.ApprovalInstance) (*uuid.UUID, error) {
	tx, err := db.Conn.Begin(db.DBCtx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())

	// Serialize ticket generation per tenant and year. The lock is released when
	// this transaction commits or rolls back.
	requestDate := time.Now()
	lockKey := fmt.Sprintf("approval-ticket:%s:%d", instance.TenantUUID, requestDate.Year())
	if _, err = tx.Exec(db.DBCtx, `select pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
		return nil, err
	}

	var institutionCode string
	if err = tx.QueryRow(db.DBCtx,
		`select code from user_sch.tenant where uuid = $1`,
		instance.TenantUUID,
	).Scan(&institutionCode); err != nil {
		return nil, fmt.Errorf("get institution code: %w", err)
	}

	var nextSequence int
	if err = tx.QueryRow(db.DBCtx, `
		select coalesce(max(
			case
				when split_part(ticket_number, '/', 1) ~ '^[0-9]+$'
				then split_part(ticket_number, '/', 1)::integer
			end
		), 0) + 1
		from approval.approval_instance
		where tenant_uuid = $1
		  and split_part(ticket_number, '/', 5) = $2
	`, instance.TenantUUID, fmt.Sprintf("%d", requestDate.Year())).Scan(&nextSequence); err != nil {
		return nil, fmt.Errorf("get next ticket sequence: %w", err)
	}

	ticketNumber, err := GenerateTicketNumber(
		nextSequence,
		APPROVAL_DOCUMENT_CODE,
		CLASS_MODULE_CODE,
		institutionCode,
		requestDate,
	)
	if err != nil {
		return nil, err
	}

	query := `
	insert into approval.approval_instance 
		(
			approval_workflow_uuid,
			tenant_uuid,
			ticket_number,
			entity_type,
			entity_uuid,
			action_code,
			request_data,
			current_step,
			status_uuid,
			requested_by,
			requested_date,
			finalized_by,
			finalized_date,
			updated_date
		) values (
			$1::uuid, 
			$2::uuid, 
			$3,
			$4, 
			$5, 
			$6, 
			$7::jsonb, 
			1, 
			$8::uuid, 
			$9::uuid,
			$10,
			$11,
			$12,
			$13
		)
		returning uuid;
	`
	var resultUUID uuid.UUID
	err = tx.QueryRow(db.DBCtx, query,
		instance.ApprovalWorkflowUUID,
		instance.TenantUUID,
		ticketNumber,
		instance.EntityType,
		instance.EntityUUID,
		instance.ActionCode,
		instance.RequestData,
		instance.StatusUUID,
		instance.RequestedBy,
		requestDate,
		instance.FinalizedBy,
		instance.FinalizedDate,
		instance.UpdatedDate,
	).Scan(&resultUUID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(db.DBCtx); err != nil {
		return nil, err
	}
	return &resultUUID, nil
}

func CreateApprovalAction(action model.ApprovalAction) (*uuid.UUID, error) {
	tx, err := db.Conn.Begin(db.DBCtx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())

	query := `
		insert into approval.approval_action (
			approval_instance_uuid,
			approval_step_uuid,
			action_code,
			acted_by,
			note,
			created_date
		) values(
			$1::uuid,
			$2::uuid,
			$3,
			$4::uuid,
			$5,
			$6
		)
		returning uuid;
	`

	resultUUID, err := db.InsertReturnUUID(query,
		action.ApprovalInstanceUUID,
		action.ApprovalStepUUID,
		action.ActionCode,
		action.ActedBy,
		action.Note,
		action.CreatedDate,
	)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return nil, err
		}
	}

	return resultUUID, tx.Commit(db.DBCtx)
}

func GenerateTicketNumber(sequence int, documentCode, moduleCode, institutionCode string, requestDate time.Time) (string, error) {
	if sequence < 1 {
		return "", errors.New("ticket sequence must be greater than zero")
	}

	documentCode = strings.ToUpper(strings.TrimSpace(documentCode))
	moduleCode = strings.ToUpper(strings.TrimSpace(moduleCode))
	institutionCode = strings.ToUpper(strings.TrimSpace(institutionCode))
	if documentCode == "" || moduleCode == "" || institutionCode == "" {
		return "", errors.New("ticket codes cannot be empty")
	}
	if strings.Contains(documentCode, "/") || strings.Contains(moduleCode, "/") || strings.Contains(institutionCode, "/") {
		return "", errors.New("ticket codes cannot contain a slash")
	}
	if requestDate.IsZero() {
		return "", errors.New("ticket request date is required")
	}

	romanMonths := [...]string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}
	return fmt.Sprintf(
		"%06d/%s-%s/%s/%s/%d",
		sequence,
		documentCode,
		moduleCode,
		institutionCode,
		romanMonths[int(requestDate.Month())-1],
		requestDate.Year(),
	), nil
}

func MapIntoStuct[T any](source map[string]interface{}) (*T, error) {
	var result T

	// Normalize keys to the target's JSON field names. Approval payloads may
	// contain either Go field names (CreatedDate) or JSON names (created_date).
	// Keeping string representations intact lets encoding/json invoke the
	// standard unmarshallers implemented by uuid.UUID and time.Time.
	normalized := make(map[string]any, len(source))
	targetType := reflect.TypeOf(result)
	if targetType.Kind() != reflect.Struct {
		return &result, fmt.Errorf("MapIntoStuct target must be a struct, got %s", targetType.Kind())
	}

	for sourceKey, value := range source {
		destinationKey := sourceKey
		for i := 0; i < targetType.NumField(); i++ {
			field := targetType.Field(i)
			jsonKey := strings.Split(field.Tag.Get("json"), ",")[0]
			if jsonKey == "-" {
				continue
			}
			if jsonKey == "" {
				jsonKey = field.Name
			}
			if strings.EqualFold(sourceKey, field.Name) || strings.EqualFold(sourceKey, jsonKey) {
				destinationKey = jsonKey
				break
			}
		}
		normalized[destinationKey] = value
	}

	bytes, err := json.Marshal(normalized)
	if err != nil {
		return &result, err
	}

	err = json.Unmarshal(bytes, &result)
	return &result, err
}
