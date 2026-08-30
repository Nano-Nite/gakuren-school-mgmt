package helper

import (
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
			concat(split_part(ai.ticket_number, '/', 2), '-', split_part(ai.ticket_number, '/', 1)) as ticket_number,
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
		param = append(param, payload.Filter.Status)
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
