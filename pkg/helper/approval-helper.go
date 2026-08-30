package helper

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
)

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
			when COALESCE(datas.current_step,progress.instance_current_step) = progress.step_order then 'current'
			when COALESCE(datas.current_step,progress.instance_current_step) < progress.step_order  then 'future'
			when COALESCE(datas.current_step,progress.instance_current_step) > progress.step_order then 'past'
			when datas.action_code = 'SUBMIT' then 'init'
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
	tx, err := db.Conn.Begin(db.DBCtx)
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
	if err = db.ExecuteQuery(query, selectedStatus.UUID, uuid); err != nil {
		return err
	}

	return tx.Commit(db.DBCtx)
}

func InsertApprovalAction(payload model.ApprovalAction) error {
	tx, err := db.Conn.Begin(db.DBCtx)
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
	if err = db.ExecuteQuery(query,
		payload.ApprovalInstanceUUID,
		payload.ApprovalStepUUID,
		payload.ActionCode,
		payload.ActedBy,
		payload.Note); err != nil {
		return err
	}

	return tx.Commit(db.DBCtx)
}
