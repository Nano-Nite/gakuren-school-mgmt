package helper

import (
	"strconv"
	"strings"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
)

func SearchGender(tenantUUID string, payload model.SearchPayload) ([]model.ReadGenderResult, *model.DataStatistics, error) {
	var param []interface{}

	//* base query
	query := `
	with datas as(
		select
			g."uuid" 
			,g."name" 
			,g.abbr_name 
			,s."name" as status
		from public.gender g
		join public.status s on g.status_uuid = s.uuid
	)
	`

	// param = append(param, tenantUUID)
	queryBuilder := ""

	//* build query by payload data
	// search
	queryBuilder += `(lower(name) LIKE lower($` + strconv.Itoa(len(param)+1) + `) ` +
		` or lower(abbr_name) LIKE lower($` + strconv.Itoa(len(param)+1) + `) ` +
		`)`
	if payload.Search != nil && len(*payload.Search) > 0 {
		param = append(param, "%"+*payload.Search+"%")
	} else {
		param = append(param, "%"+""+"%")
	}

	// filter
	if payload.Filter != nil {
		if (*payload.Filter)["status"] != nil {
			queryBuilder += ` and lower(status) = lower($` + strconv.Itoa(len(param)+1) + `)`
			param = append(param, (*payload.Filter)["status"].(string))
		}
		if (*payload.Filter)["uuid"] != nil {
			queryBuilder += ` and datas.uuid = $` + strconv.Itoa(len(param)+1)
			param = append(param, (*payload.Filter)["uuid"].(string))
		}
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
		query += `SELECT * FROM datas WHERE ` + queryBuilder
	}

	selectedData, err := db.GetMultipleDataByQuery[model.ReadGenderResult](query, param...)
	if err != nil {
		return nil, nil, err
	}

	dataStat := CalculateDataStatisticResult(count, payload, len(*selectedData))

	return *selectedData, &dataStat, nil
}
