package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrApprovalFinalized       = errors.New("approval already finalized")
	ErrApprovalUnauthorized    = errors.New("user is not allowed to execute this approval")
	ErrApprovalDuplicateAction = errors.New("user has already acted on this approval step")
)

// ExecuteApproval records an approval decision and updates the instance in one
// transaction. The instance row is locked so two approvers cannot advance the
// same step independently.
func ExecuteApproval(instanceUUID, schoolUUID, tenantUUID string, actedBy, roleUUID uuid.UUID, command string, note *string) (bool, error) {
	command = strings.ToUpper(strings.TrimSpace(command))
	if command != ACTION_CODE_CANCEL && command != ACTION_CODE_REJECT && command != ACTION_CODE_APPROVE {
		return false, fmt.Errorf("unsupported approval command: %s", command)
	}

	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.Background())

	var workflowUUID, requestedBy, statusUUID uuid.UUID
	var instanceEntityUUID *uuid.UUID
	var currentStep int
	var entityType, instanceAction string
	var requestData json.RawMessage

	parsedTenantUUID, err := uuid.Parse(tenantUUID)
	if err != nil {
		return false, fmt.Errorf("invalid tenant UUID: %w", err)
	}
	parsedSchoolUUID, err := uuid.Parse(schoolUUID)
	if err != nil {
		return false, fmt.Errorf("invalid school UUID: %w", err)
	}

	// lock the approval instance row for update
	err = tx.QueryRow(context.Background(), `
		select approval_workflow_uuid, requested_by, current_step, status_uuid,
		       entity_type, entity_uuid, action_code, request_data
		from approval.approval_instance
		where uuid = $1 and tenant_uuid = $2
		for update
	`, instanceUUID, tenantUUID).Scan(
		&workflowUUID, &requestedBy, &currentStep, &statusUUID,
		&entityType, &instanceEntityUUID, &instanceAction, &requestData,
	)
	if err != nil {
		return false, err
	}

	// check if the approval instance is still active
	var statusName string
	if err = tx.QueryRow(context.Background(), `select name from public.status where uuid = $1`, statusUUID).Scan(&statusName); err != nil {
		return false, err
	}
	if !strings.EqualFold(statusName, STATUS_ACTIVE) {
		return false, ErrApprovalFinalized
	}

	// get the current step details
	var stepUUID uuid.UUID
	var approverRoleUUID uuid.UUID
	var requiredApprovals int
	err = tx.QueryRow(context.Background(), `
		select uuid, approver_role_uuid, required_approvals
		from approval.approval_step
		where approval_workflow_uuid = $1 and step_order = $2
	`, workflowUUID, currentStep).Scan(&stepUUID, &approverRoleUUID, &requiredApprovals)
	if err != nil {
		return false, err
	}

	var actionStepUUID *uuid.UUID
	if command == ACTION_CODE_CANCEL {
		if requestedBy != actedBy || currentStep != 1 {
			return false, ErrApprovalUnauthorized
		}
	} else {
		if approverRoleUUID != roleUUID {
			return false, ErrApprovalUnauthorized
		}
		actionStepUUID = &stepUUID

		var alreadyActed bool
		if err = tx.QueryRow(context.Background(), `
			select exists (
				select 1 from approval.approval_action
				where approval_instance_uuid = $1
				  and approval_step_uuid = $2
				  and acted_by = $3
				  and action_code in ($4, $5)
			)
		`, instanceUUID, stepUUID, actedBy, ACTION_CODE_APPROVE, ACTION_CODE_REJECT).Scan(&alreadyActed); err != nil {
			return false, err
		}
		if alreadyActed {
			return false, ErrApprovalDuplicateAction
		}
	}

	// insert approval action record
	_, err = tx.Exec(context.Background(), `
		insert into approval.approval_action
			(approval_instance_uuid, approval_step_uuid, action_code, acted_by, note, created_date)
		values ($1, $2, $3, $4, $5, now())
	`, instanceUUID, actionStepUUID, command, actedBy, note)
	if err != nil {
		return false, err
	}

	// check if the approval instance is finalized
	finalized := command == ACTION_CODE_CANCEL || command == ACTION_CODE_REJECT
	if command == ACTION_CODE_APPROVE {
		var approvalCount int
		if err = tx.QueryRow(context.Background(), `
			select count(distinct acted_by)
			from approval.approval_action
			where approval_instance_uuid = $1
			  and approval_step_uuid = $2
			  and action_code = $3
		`, instanceUUID, stepUUID, ACTION_CODE_APPROVE).Scan(&approvalCount); err != nil {
			return false, err
		}

		if approvalCount >= requiredApprovals {
			var hasNextStep bool
			if err = tx.QueryRow(context.Background(), `
				select exists (
					select 1 from approval.approval_step
					where approval_workflow_uuid = $1 and step_order = $2
				)
			`, workflowUUID, currentStep+1).Scan(&hasNextStep); err != nil {
				return false, err
			}
			if hasNextStep {
				_, err = tx.Exec(context.Background(), `
					update approval.approval_instance
					set current_step = current_step + 1, updated_date = now()
					where uuid = $1
				`, instanceUUID)
			} else {
				finalized = true
			}
		}
	}
	if err != nil {
		return false, err
	}

	if finalized {
		var entityUUID *uuid.UUID
		switch command {
		case ACTION_CODE_APPROVE:
			switch {
			//* Class CRUD
			//create
			case strings.EqualFold(entityType, CLASS_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_CREATE):
				// Decode UUIDs as text first so an empty optional UUID does not make
				// json.Unmarshal reject the entire approval payload.
				mapData := make(map[string]interface{})
				if err = json.Unmarshal(requestData, &mapData); err != nil {
					return false, fmt.Errorf("decode class approval request: %w", err)
				}

				classData, err := MapIntoStuct[model.ClassModel](mapData)
				if err != nil {
					return false, fmt.Errorf("convert class approval request: %w", err)
				}
				err = InsertClass(*classData, parsedSchoolUUID, parsedTenantUUID)
				if err != nil {
					return false, fmt.Errorf("create approved class: %w", err)
				}

				entityUUID = classData.UUID
			// update
			case strings.EqualFold(entityType, CLASS_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_UPDATE):
				if instanceEntityUUID == nil {
					return false, errors.New("class update approval is missing entity UUID")
				}

				mapData := make(map[string]interface{})
				if err = json.Unmarshal(requestData, &mapData); err != nil {
					return false, fmt.Errorf("decode class approval request: %w", err)
				}

				classData, convertErr := MapIntoStuct[model.ClassModel](mapData)
				if convertErr != nil {
					return false, fmt.Errorf("convert class approval request: %w", convertErr)
				}

				if err = UpdateClass(*classData, parsedSchoolUUID, parsedTenantUUID); err != nil {
					return false, fmt.Errorf("update approved class: %w", err)
				}

				classData.StatusUUID = DB_UUID_STATUS_ACTIVE
				if err = UpdateClassStatus(*classData, classData.StatusUUID, parsedTenantUUID, parsedSchoolUUID); err != nil {
					return false, fmt.Errorf("failed to update class status: %w", err)
				}
				entityUUID = instanceEntityUUID
			// delete
			case strings.EqualFold(entityType, CLASS_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_DELETE):
				if instanceEntityUUID == nil {
					return false, errors.New("class delete approval is missing entity UUID")
				}

				var data model.ClassModel
				if err = json.Unmarshal(requestData, &data); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}

				if err = SoftDeleteClass(data, parsedTenantUUID, parsedSchoolUUID); err != nil {
					return false, fmt.Errorf("soft-delete approved class: %w", err)
				}

				entityUUID = instanceEntityUUID
			// activate
			case strings.EqualFold(entityType, CLASS_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_ACTIVATE):
				if instanceEntityUUID == nil {
					return false, errors.New("class activate approval is missing entity UUID")
				}

				mapData := make(map[string]interface{})
				if err = json.Unmarshal(requestData, &mapData); err != nil {
					return false, fmt.Errorf("decode class approval request: %w", err)
				}

				classData, convertErr := MapIntoStuct[model.ClassModel](mapData)
				if convertErr != nil {
					return false, fmt.Errorf("convert class approval request: %w", convertErr)
				}

				if err = UpdateClassStatus(*classData, DB_UUID_STATUS_ACTIVE, parsedTenantUUID, parsedSchoolUUID); err != nil {
					return false, fmt.Errorf("update approved class: %w", err)
				}

				entityUUID = instanceEntityUUID

			//* Student CRUD
			// create
			case strings.EqualFold(entityType, STUDENT_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_CREATE):
				var studenData model.CreateStudentModel
				if err = json.Unmarshal(requestData, &studenData); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}
				parsedTenantUUID, err := uuid.Parse(tenantUUID)
				if err != nil {
					return false, fmt.Errorf("invalid tenant UUID: %w", err)
				}
				studentRole, err := GetRoleByAbbrName(ROLE_STUDENT)
				if err != nil {
					return false, fmt.Errorf("role not found: %w", err)
				}

				userData := model.UserModel{
					TenantUUID: parsedTenantUUID,
					Name:       studenData.Name,
					Email:      studenData.Email,
					Phone:      studenData.Phone,
					Address:    studenData.Address,
					RoleUUID:   studentRole.UUID,
					StatusUUID: DB_UUID_STATUS_ACTIVE,
				}

				id, insertErr := InsertUserStudent(userData)
				if insertErr != nil {
					return false, nil
				}

				createUUID, err := InsertStudent(studenData, *id, DB_UUID_STATUS_ACTIVE)
				if err != nil {
					return false, nil
				}

				entityUUID = createUUID
			//update
			case strings.EqualFold(entityType, STUDENT_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_UPDATE):
				if instanceEntityUUID == nil {
					return false, errors.New("user update approval is missing entity UUID")
				}
				var userData model.StudentModel
				if err = json.Unmarshal(requestData, &userData); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}

				// update user tabel first
				err = tx.QueryRow(context.Background(), `
					update user_sch."user" set 
						name=$1 , phone=$2, address=$3, updated_date=now()
					where uuid=$5 and tenant_uuid=$6 returning uuid
				`, userData.Name, userData.Phone, userData.Address, userData.UserUUID, tenantUUID).Scan(instanceEntityUUID)
				if errors.Is(err, pgx.ErrNoRows) {
					return false, errors.New("approved user update target not found")
				}
				if err != nil {
					return false, fmt.Errorf("update approved user: %w", err)
				}

				// then update student table
				err = tx.QueryRow(context.Background(), `
					update school_sch.student set 
						gender_uuid=$1, class_uuid=$2, nis=$3, nisn=$4, status_uuid =$5, updated_date=now()
					where uuid=$6 returning uuid
				`, userData.GenderUUID, userData.ClassUUID, userData.NIS, userData.NISN, userData.StatusUUID, userData.UUID).Scan(instanceEntityUUID)
				if errors.Is(err, pgx.ErrNoRows) {
					return false, errors.New("approved student update target not found")
				}
				if err != nil {
					return false, fmt.Errorf("update approved user: %w", err)
				}

				entityUUID = instanceEntityUUID
			// delete
			case strings.EqualFold(entityType, STUDENT_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_DELETE):
				if instanceEntityUUID == nil {
					return false, errors.New("student delete approval is missing entity UUID")
				}
				var userData model.StudentModel
				if err = json.Unmarshal(requestData, &userData); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}
				// update user tabel first
				err = tx.QueryRow(context.Background(), `
					update user_sch."user" set 
						status_uuid=$1, updated_date=now()
					where uuid=$2 and tenant_uuid=$3 returning uuid
				`, DB_UUID_STATUS_DELETED, userData.UserUUID, tenantUUID).Scan(instanceEntityUUID)
				if errors.Is(err, pgx.ErrNoRows) {
					return false, errors.New("approved user update target not found")
				}
				if err != nil {
					return false, fmt.Errorf("update approved student: %w", err)
				}

				// then update student table
				err = tx.QueryRow(context.Background(), `
					update school_sch.student set 
						status_uuid =$1, updated_date=now()
					where uuid=$2 returning uuid
				`, DB_UUID_STATUS_DELETED, userData.UUID).Scan(instanceEntityUUID)
				if errors.Is(err, pgx.ErrNoRows) {
					return false, errors.New("approved student delete target not found")
				}
				if err != nil {
					return false, fmt.Errorf("delete approved user: %w", err)
				}
				entityUUID = instanceEntityUUID

			//* Teacher and Staff CRUD
			// create
			case strings.EqualFold(entityType, TNS_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_CREATE):
				var data model.CreateTNSModel
				if err = json.Unmarshal(requestData, &data); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}
				parsedTenantUUID, err := uuid.Parse(tenantUUID)
				if err != nil {
					return false, fmt.Errorf("invalid tenant UUID: %w", err)
				}
				parsedSchoolUUID, err := uuid.Parse(schoolUUID)
				if err != nil {
					return false, fmt.Errorf("invalid school UUID: %w", err)
				}

				var selectedRole model.RoleModel
				if data.IsStaff {
					role, err := GetRoleByAbbrName(ROLE_STAFF)
					if err != nil {
						return false, fmt.Errorf("role not found: %w", err)
					}
					selectedRole = *role
				} else {
					role, err := GetRoleByAbbrName(ROLE_TEACHER)
					if err != nil {
						return false, fmt.Errorf("role not found: %w", err)
					}
					selectedRole = *role
				}

				userData := model.UserModel{
					TenantUUID: parsedTenantUUID,
					Name:       &data.Biodata.Fullname,
					Email:      &data.Biodata.Email,
					Phone:      &data.Biodata.Phone,
					Address:    &data.Biodata.Address,
					RoleUUID:   selectedRole.UUID,
					StatusUUID: DB_UUID_STATUS_NEWUSER,
				}

				id, insertErr := InsertTNS(data, userData, parsedTenantUUID, parsedSchoolUUID)
				if insertErr != nil {
					return false, nil
				}

				entityUUID = id
			//update
			case strings.EqualFold(entityType, TNS_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_UPDATE):
				if instanceEntityUUID == nil {
					return false, errors.New("user update approval is missing entity UUID")
				}
				var data model.UpdateTNSModel
				if err = json.Unmarshal(requestData, &data); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}

				// update user and employee table
				if err = UpdateTNS(data, parsedTenantUUID, parsedSchoolUUID); err != nil {
					return false, fmt.Errorf("update Teacher or Staff: %w", err)
				}

				// update user status
				if err = UpdateTNSStatus(data, parsedSchoolUUID, parsedTenantUUID, DB_UUID_STATUS_ACTIVE); err != nil {
					return false, fmt.Errorf("update Teacher or Staff: %w", err)
				}

				entityUUID = instanceEntityUUID
			// delete
			case strings.EqualFold(entityType, TNS_ENTITY_TYPE) && strings.EqualFold(instanceAction, ACTION_CODE_DELETE):
				if instanceEntityUUID == nil {
					return false, errors.New("teacher and staff delete approval is missing entity UUID")
				}
				var data model.UpdateTNSModel
				if err = json.Unmarshal(requestData, &data); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}

				err := SoftDeleteTNS(data, parsedTenantUUID, parsedSchoolUUID)
				if err != nil {
					return false, fmt.Errorf("delete approval teacher or staff: %w", err)
				}

				entityUUID = instanceEntityUUID

			default:
				return false, fmt.Errorf("unsupported approved entity/action: %s/%s", entityType, instanceAction)
			}
		case ACTION_CODE_CANCEL, ACTION_CODE_REJECT: // cancel or reject
			switch {
			//* Class
			case strings.EqualFold(entityType, CLASS_ENTITY_TYPE):
				// in case create data but canceled at time
				if instanceEntityUUID == nil {
					break
				}

				var data model.ClassModel
				if err = json.Unmarshal(requestData, &data); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}

				err = tx.QueryRow(context.Background(), `
					update school_sch.class
					set status_uuid = $1, updated_date = now()
					where uuid = $2 and tenant_uuid = $3 and school_uuid = $4
					returning uuid
				`, data.StatusUUID, instanceEntityUUID, tenantUUID, schoolUUID).Scan(instanceEntityUUID)
				if errors.Is(err, pgx.ErrNoRows) {
					return false, errors.New("cancel class target not found")
				}
				if err != nil {
					return false, fmt.Errorf("canceled approval class: %w", err)
				}
				entityUUID = instanceEntityUUID

			//* Student
			case strings.EqualFold(entityType, STUDENT_ENTITY_TYPE):
				// in case create data but canceled at time
				if instanceEntityUUID == nil {
					break
				}

				var data model.StudentModel
				if err = json.Unmarshal(requestData, &data); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}

				if data.Activate {
					err = tx.QueryRow(context.Background(), `update school_sch.student set status_uuid=$1,updated_date=now()
						where uuid=$2 returning uuid`, DB_UUID_STATUS_INACTIVE, instanceEntityUUID).Scan(instanceEntityUUID)
					if errors.Is(err, pgx.ErrNoRows) {
						return false, errors.New("cancel user target not found")
					}
					if err != nil {
						return false, fmt.Errorf("cancel user approval: %w", err)
					}
				} else {
					err = tx.QueryRow(context.Background(), `update school_sch.student set status_uuid=$1,updated_date=now()
						where uuid=$2 returning uuid`, DB_UUID_STATUS_ACTIVE, instanceEntityUUID).Scan(instanceEntityUUID)
					if errors.Is(err, pgx.ErrNoRows) {
						return false, errors.New("cancel user target not found")
					}
					if err != nil {
						return false, fmt.Errorf("cancel user approval: %w", err)
					}
				}
				entityUUID = instanceEntityUUID

			//* TNS
			case strings.EqualFold(entityType, TNS_ENTITY_TYPE):
				// in case create data but canceled at time
				if instanceEntityUUID == nil {
					break
				}

				var data model.UpdateTNSModel
				if err = json.Unmarshal(requestData, &data); err != nil {
					return false, fmt.Errorf("decode user approval request: %w", err)
				}

				if data.Activate {
					err = tx.QueryRow(context.Background(), `update user_sch.user set status_uuid=$1,updated_date=now()
						where uuid=$2 returning uuid`, DB_UUID_STATUS_INACTIVE, instanceEntityUUID).Scan(instanceEntityUUID)
					if errors.Is(err, pgx.ErrNoRows) {
						return false, errors.New("cancel user target not found")
					}
					if err != nil {
						return false, fmt.Errorf("cancel user approval: %w", err)
					}
				} else {
					err = tx.QueryRow(context.Background(), `update user_sch.user set status_uuid=$1,updated_date=now()
						where uuid=$2 returning uuid`, DB_UUID_STATUS_ACTIVE, instanceEntityUUID).Scan(instanceEntityUUID)
					if errors.Is(err, pgx.ErrNoRows) {
						return false, errors.New("cancel user target not found")
					}
					if err != nil {
						return false, fmt.Errorf("cancel user approval: %w", err)
					}
				}
				entityUUID = instanceEntityUUID
			}
		default:
			return false, fmt.Errorf("unsupported approved entity/action: %s/%s", entityType, instanceAction)
		}

		statusCandidates := []string{command}
		switch command {
		case ACTION_CODE_APPROVE:
			statusCandidates = append(statusCandidates, "APPROVED")
		case ACTION_CODE_REJECT:
			statusCandidates = append(statusCandidates, "REJECTED")
		case ACTION_CODE_CANCEL:
			statusCandidates = append(statusCandidates, "CANCELLED", "CANCELED")
		}

		var finalStatusUUID uuid.UUID
		if err = tx.QueryRow(context.Background(), `
			select uuid
			from public.status
			where lower(name) = any($1)
			order by array_position($1, lower(name))
			limit 1
		`, lowerStrings(statusCandidates)).Scan(&finalStatusUUID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false, fmt.Errorf("no final status is configured for command %q", command)
			}
			return false, err
		}
		_, err = tx.Exec(context.Background(), `
			update approval.approval_instance
			set status_uuid = $1, finalized_by = $2, finalized_date = now(),
			    updated_date = now(), entity_uuid = coalesce($3, entity_uuid)
			where uuid = $4
		`, finalStatusUUID, actedBy, entityUUID, instanceUUID)
		if err != nil {
			return false, err
		}
	}

	if err = tx.Commit(context.Background()); err != nil {
		return false, err
	}
	return finalized, nil
}

func lowerStrings(values []string) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = strings.ToLower(value)
	}
	return result
}

func MyApproval(uuid string, roleUUID string, payload model.SearchPayload) ([]model.MyApprovalResult, *model.DataStatistics, error) {
	var param []interface{}

	//* base query
	query := `
	WITH datas AS (
		SELECT
			ai.uuid,
			aw.uuid AS workflow_uuid,
			aw.name AS workflow_name,
			ai.ticket_number,
			ai.current_step,
			s2.name AS status,
			s."name" AS requested_by,
			r.name AS role_name,
			ai.requested_date
		FROM approval.approval_instance ai
		JOIN approval.approval_workflow aw ON aw.uuid = ai.approval_workflow_uuid
		JOIN approval.approval_step aps ON aps.approval_workflow_uuid = ai.approval_workflow_uuid AND aps.step_order = ai.current_step
		JOIN user_sch.user s ON ai.requested_by = s.uuid
		JOIN user_sch.role r ON s.role_uuid = r.uuid
		JOIN public.status s2 ON ai.status_uuid = s2.uuid
		WHERE ai.requested_by = $1 OR aps.approver_role_uuid = $2
	)
	`

	param = append(param, uuid, roleUUID)
	queryBuilder := ""

	//* build query by payload data
	// search
	queryBuilder += `(lower(workflow_name) LIKE $` + strconv.Itoa(len(param)+1) +
		` or lower(ticket_number) LIKE $` + strconv.Itoa(len(param)+1) +
		` or lower(requested_by) LIKE $` + strconv.Itoa(len(param)+1) +
		` or lower(role_name) LIKE $` + strconv.Itoa(len(param)+1) +
		`)`
	if payload.Search != nil && len(*payload.Search) > 0 {
		param = append(param, "%"+*payload.Search+"%")
	} else {
		param = append(param, "%"+""+"%")
	}

	// filter
	if payload.Filter != nil {
		queryBuilder += ` and lower(status) = lower($` + strconv.Itoa(len(param)+1) + `)`
		param = append(param, (*payload.Filter)["status"])
	}

	// run count first to get data statistic
	queryCount := query + "SELECT COUNT(*) FROM datas WHERE " + queryBuilder
	count, err := db.GetSingleDataByQuery[model.CountResult](queryCount, param...)
	if err != nil {
		return nil, nil, err
	}

	// order by
	if payload.SortBy != nil {
		queryBuilder += ` ORDER BY `
		for i, sortBy := range *payload.SortBy {
			for key, value := range sortBy {
				if strings.ToLower(value.(string)) == "asc" || strings.ToLower(value.(string)) == "desc" {
					queryBuilder += key + ` ` + value.(string)
					if i+1 < len(*payload.SortBy) {
						queryBuilder += `, `
					}
				}
			}
		}
	}

	// limit
	if payload.RowPerPage != nil && *payload.RowPerPage != 0 {
		queryBuilder += ` LIMIT $` + strconv.Itoa(len(param)+1)
		param = append(param, *payload.RowPerPage)
	} else {
		queryBuilder += ` LIMIT $` + strconv.Itoa(len(param)+1)
		param = append(param, DEFAULT_ROW_PER_PAGES)
	}

	// offset
	if payload.Page != nil && *payload.Page != 0 {
		queryBuilder += ` OFFSET $` + strconv.Itoa(len(param)+1)
		if payload.RowPerPage != nil && *payload.RowPerPage != 0 {
			param = append(param, *payload.Page**payload.RowPerPage-*payload.RowPerPage)
		} else {
			param = append(param, *payload.Page*DEFAULT_ROW_PER_PAGES-DEFAULT_ROW_PER_PAGES)
		}
	} else {
		queryBuilder += ` OFFSET $` + strconv.Itoa(len(param)+1)
		param = append(param, DEFAULT_PAGES*DEFAULT_ROW_PER_PAGES-DEFAULT_ROW_PER_PAGES)
	}

	if len(queryBuilder) > 0 {
		query += `SELECT 
			datas.*
			,(select count(*) as total_step from approval.approval_step t where t.approval_workflow_uuid = datas.workflow_uuid)
		FROM datas WHERE ` + queryBuilder
	}

	selectedData, err := db.GetMultipleDataByQuery[model.MyApprovalResult](query, param...)
	if err != nil {
		return nil, nil, err
	}

	dataStat := CalculateDataStatisticResult(count, payload, len(*selectedData))

	return *selectedData, &dataStat, nil
}

func DetailApproval(uuid string, tenantUUID string) (*model.DetailApprovalModel, error) {
	query := `
	with datas as(
		SELECT
			ai.uuid,
			aw.uuid as workflow_uuid,
			aw.name AS workflow_name,
			ai.ticket_number,
			ai.current_step,
			s2.name as status,
			s."name" requested_by,
			r.name role_name,
			ai.requested_date,
			ai.entity_type,
			ai.entity_uuid,
			ai.request_data,
			ai.action_code 
		FROM approval.approval_instance ai
		JOIN approval.approval_workflow aw ON aw.uuid = ai.approval_workflow_uuid
		JOIN approval.approval_step aps ON aps.approval_workflow_uuid = ai.approval_workflow_uuid AND aps.step_order = ai.current_step
		join user_sch.user s on ai.requested_by = s.uuid
		join user_sch.role r on s.role_uuid = r.uuid
		join public.status s2 on ai.status_uuid = s2.uuid
		WHERE ai.tenant_uuid = $2
	)
	select 
		datas.*
		,(select count(*) as total_step from approval.approval_step t where t.approval_workflow_uuid = datas.workflow_uuid)
	from datas 
	where datas.uuid = $1
	`
	selectedInstance, err := db.GetSingleDataByQuery[model.DetailApprovalHeader](query, uuid, tenantUUID)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return nil, err
		}
	}
	if selectedInstance == nil {
		return nil, errors.New("Workflow instance Not  Found")
	}

	query = `
	with datas as(
		select
			ai."uuid" 
			,ai.approval_workflow_uuid 
			,aa.approval_step_uuid 
			,aa.created_date as approve_date
			,aa.note 
			,aa.action_code 
			,ai.current_step
			,aa.acted_by 
			,aa.created_date 
			,(select r.name from user_sch.role r where r.uuid = s.role_uuid ) as role_name
		from approval.approval_instance ai 
		join approval.approval_action aa on aa.approval_instance_uuid = ai."uuid" 
		left join user_sch.user s on aa.acted_by = s.uuid
		where ai."uuid" = $1
	), progress as (
		select 
			ai."uuid" as instance_uuid
			,ai.current_step as instance_current_step
			,t.*
		from approval.approval_instance ai 
		join approval.approval_step t on ai.approval_workflow_uuid = t.approval_workflow_uuid 
		where ai."uuid" = $1
	)
	select 
		COALESCE(datas.action_code, 'PENDING') AS action_code
		,case 
			when datas.action_code = 'SUBMIT' then 'past'
			when datas.action_code in ('APPROVE', 'REJECT', 'CANCEL') then 'past'
			when COALESCE(datas.current_step,progress.instance_current_step) = progress.step_order then 'current'
			when COALESCE(datas.current_step,progress.instance_current_step) < progress.step_order  then 'future'
			when COALESCE(datas.current_step,progress.instance_current_step) > progress.step_order then 'past'
			else 'past'
		end as state
		,COALESCE(datas.role_name, r."name") AS role_name
		,s."name" as act_by
		,datas.created_date as approve_date
		,datas.note
	from datas
	full join progress on datas.approval_workflow_uuid = progress.approval_workflow_uuid and datas.approval_step_uuid = progress.uuid
	left join user_sch.role r on progress.approver_role_uuid = r.uuid
	left join user_sch.user s on datas.acted_by = s.uuid
	order by datas.approve_date asc, step_order asc
	`
	selectedProgress, err := db.GetMultipleDataByQuery[model.DetailApprovalProgress](query, uuid)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return nil, err
		}
	}
	if selectedProgress == nil {
		return nil, errors.New("Workflow instance Not  Found")
	}

	var result model.DetailApprovalModel
	result.DetailApprovalHeader = *selectedInstance
	result.DetailApprovalProgress = *selectedProgress

	return &result, nil
}

func UpdateApprovalInstancteStatus(uuid string, status string) error {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	//* get Active Status
	selectedStatus, err := GetStatusByName(status)
	if err != nil {
		return err
	}
	query := `UPDATE approval.approval_instance SET status_uuid = $1, updated_date = now() WHERE uuid = $2`
	if err = db.ExecuteQuery(context.Background(), query, selectedStatus.UUID, uuid); err != nil {
		return err
	}

	return tx.Commit(context.Background())
}

func InsertApprovalAction(payload model.ApprovalAction) error {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	query := `
		insert into approval.approval_action (
			approval_instance_uuid,
			approval_step_uuid,
			action_code,
			acted_by,
			note
		) values ($1, $2, $3, $4, $5);
	`
	if err = db.ExecuteQuery(context.Background(), query,
		payload.ApprovalInstanceUUID,
		payload.ApprovalStepUUID,
		payload.ActionCode,
		payload.ActedBy,
		payload.Note); err != nil {
		return err
	}

	return tx.Commit(context.Background())
}
