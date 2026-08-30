package helper

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
)

func InsertClass(data model.ClassModel) error {
	tx, err := db.Conn.Begin(db.DBCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	query := `SELECT * FROM school_sch.class WHERE lower(name) = lower($1) or lower(abbr_name) = lower($2) LIMIT 1`
	selectedClass, err := db.GetSingleDataByQuery[model.ClassModel](query, data.Name, data.AbbrName)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return err
		}
	}

	if selectedClass != nil {
		return errors.New("class already exist")
	}

	query = `INSERT INTO school_sch.class (name, abbr_name, level, homeroom_teacher,status_uuid, created_date, updated_date, tenant_uuid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = db.InsertReturnUUID(query, data.Name, data.AbbrName, data.Level, data.HomeroomTeacher, data.StatusUUID, data.CreatedDate, data.UpdatedDate, data.TenantUUID)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return err
		}
	}

	return tx.Commit(db.DBCtx)
}

func SearchClass(tenantUUID string, payload model.SearchPayload) ([]model.ReadClassModelResult, *model.DataStatistics, error) {
	var param []interface{}

	//* base query
	query := `
	with datas as(
		select
			c.uuid,
			c.name,
			c.abbr_name,
			c.level,
			u.name as homeroom_teacher,
			s.name as status,
			0 as total_student
		from school_sch.class c
		left join user_sch.user u on c.homeroom_teacher = u.uuid
		left join public.status s on c.status_uuid = s.uuid
		where c.tenant_uuid = $1
	)
	`

	param = append(param, tenantUUID)
	queryBuilder := ""

	//* build query by payload data
	// search
	queryBuilder += `(lower(name) LIKE $` + strconv.Itoa(len(param)+1) +
		` or lower(abbr_name) LIKE $` + strconv.Itoa(len(param)+1) +
		` or lower(homeroom_teacher) LIKE $` + strconv.Itoa(len(param)+1) +
		` or level::text LIKE $` + strconv.Itoa(len(param)+1) +
		` or total_student::text LIKE $` + strconv.Itoa(len(param)+1) +
		`)`
	if payload.Search != nil && len(*payload.Search) > 0 {
		param = append(param, "%"+*payload.Search+"%")
	} else {
		param = append(param, "%"+""+"%")
	}

	// filter
	if payload.Filter != nil {
		// log.Println(payload.Filter)
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
		query += `SELECT * FROM datas WHERE ` + queryBuilder
	}

	selectedData, err := db.GetMultipleDataByQuery[model.ReadClassModelResult](query, param...)
	if err != nil {
		return nil, nil, err
	}

	dataStat := CalculateDataStatisticResult(count, payload, len(*selectedData))

	return *selectedData, &dataStat, nil
}
