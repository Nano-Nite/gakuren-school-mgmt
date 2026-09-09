package helper

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GetClass(classUUID, tenantUUID, schoolUUID uuid.UUID) (*model.ClassModel, error) {
	classData, err := db.GetSingleDataByQuery[model.ClassModel](`
		select uuid, name, abbr_name, level, homeroom_teacher, status_uuid,
		       created_date, updated_date, tenant_uuid
		from school_sch.class
		where uuid = $1 and tenant_uuid = $2 and school_uuid = $3
	`, classUUID, tenantUUID, schoolUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || err.Error() == "no rows in result set" {
			return nil, errors.New("class not found")
		}
		return nil, err
	}
	return classData, nil
}

func UpdateClass(data model.ClassModel, schoolUUID, tenantUUID uuid.UUID) error {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	selectedData, err := GetClass(*data.UUID, data.TenantUUID, data.SchoolUUID)
	if err != nil {
		return errors.New("class not found")
	}

	result, err := tx.Exec(context.Background(), `
		update school_sch.class
		set name = $1, abbr_name = $2, level = $3,
		    homeroom_teacher = $4, updated_date = now()
		where uuid = $5 and tenant_uuid = $6 and school_uuid = $7
	`, data.Name, data.AbbrName, data.Level, data.HomeroomTeacher, data.UUID, data.TenantUUID, data.SchoolUUID)
	if err != nil {
		return fmt.Errorf("update class: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("class not found")
	}

	// check if homeroom teacher has changed
	if selectedData.HomeroomTeacher != nil && data.HomeroomTeacher != nil && *selectedData.HomeroomTeacher != *data.HomeroomTeacher {
		// remove previous homeroom teacher position
		_, err = tx.Exec(context.Background(), `
			delete from employee.employee_position
			where employee_uuid = (select e.uuid from employee.employee e where e.user_uuid = $1)
			and position_uuid = (select uuid from public."position" p where lower(p.abbr_name ) = 'wk' limit 1)
		`, *selectedData.HomeroomTeacher)
		if err != nil {
			return fmt.Errorf("remove previous homeroom teacher role: %w", err)
		}

		// assign new homeroom teacher position
		tnsResult, err := GetTNS(data.SchoolUUID, data.TenantUUID, *data.HomeroomTeacher)
		if err != nil {
			return fmt.Errorf("get new homeroom teacher tns: %w", err)
		}
		if tnsResult == nil {
			return errors.New("new homeroom teacher not found")
		}

		_, err = tx.Exec(context.Background(), `
			insert into employee.employee_position 
			values
			($1, (select uuid from public."position" p where lower(p.abbr_name ) = 'wk' limit 1))
		`, tnsResult.EmployeeUUID)
		if err != nil {
			return fmt.Errorf("assign new homeroom teacher role: %w", err)
		}
	}

	return tx.Commit(context.Background())
}

func UpdateClassStatus(data model.ClassModel, statusUUID, tenantUUID, schoolUUID uuid.UUID) error {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	result, err := tx.Exec(context.Background(), `
		update school_sch.class
		set status_uuid = $1, updated_date = now()
		where uuid = $2 and tenant_uuid = $3 and school_uuid = $4
	`, statusUUID, data.UUID, tenantUUID, schoolUUID)
	if err != nil {
		return fmt.Errorf("update class: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("class not found")
	}
	return tx.Commit(context.Background())
}

func SoftDeleteClass(data model.ClassModel, tenantUUID, schoolUUID uuid.UUID) error {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	// update class status to inactive
	result, err := tx.Exec(context.Background(), `
		update school_sch.class
		set status_uuid = $1, updated_date = now()
		where uuid = $2 and tenant_uuid = $3 and school_uuid = $4
	`, DB_UUID_STATUS_INACTIVE, data.UUID, tenantUUID, schoolUUID)
	if err != nil {
		return fmt.Errorf("soft-delete class: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("class not found")
	}

	// remove homeroom teacher position if exists
	if data.HomeroomTeacher != nil {
		// remove homeroom teacher position
		_, err = tx.Exec(context.Background(), `
			delete from employee.employee_position
			where employee_uuid = (select e.uuid from employee.employee e where e.user_uuid = $1)
			and position_uuid = (select uuid from public."position" p where lower(p.abbr_name ) = 'wk' limit 1)
		`, *data.HomeroomTeacher)
		if err != nil {
			return fmt.Errorf("remove homeroom teacher role: %w", err)
		}

		// remove homeroom teacher in class
		_, err = tx.Exec(context.Background(), `
			update school_sch.class
			set homeroom_teacher = NULL
			where uuid = $1 and tenant_uuid = $2 and school_uuid = $3
		`, data.UUID, tenantUUID, schoolUUID)
		if err != nil {
			return fmt.Errorf("remove homeroom teacher in class: %w", err)
		}
	}

	// remove students associated with the class
	_, err = tx.Exec(context.Background(), `
		UPDATE school_sch.student
		SET class_uuid = NULL
		WHERE class_uuid = $1
	`, data.UUID)
	if err != nil {
		return fmt.Errorf("soft-delete students in class: %w", err)
	}

	return tx.Commit(context.Background())
}

func InsertClass(data model.ClassModel, schoolUUID, tenantUUID uuid.UUID) error {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	query := `select uuid, name, abbr_name, level, homeroom_teacher, status_uuid,
		created_date, updated_date, tenant_uuid
		from school_sch.class
		where tenant_uuid = $3 and (lower(name) = lower($1)
			or ($2::text is not null and lower(abbr_name) = lower($2))) limit 1`
	selectedClass, err := db.GetSingleDataByQuery[model.ClassModel](query, data.Name, data.AbbrName, data.TenantUUID)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return err
		}
	}

	if selectedClass != nil {
		return errors.New("class already exist")
	}

	query = `INSERT INTO school_sch.class (name, abbr_name, level, homeroom_teacher,status_uuid, created_date, updated_date, tenant_uuid, school_uuid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = tx.Exec(context.Background(), query, data.Name, data.AbbrName, data.Level, data.HomeroomTeacher, data.StatusUUID, data.CreatedDate, data.UpdatedDate, data.TenantUUID, data.SchoolUUID)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return err
		}
	}

	// assign homeroom teacher position to employee if homeroom teacher is not null
	if data.HomeroomTeacher != nil {
		tnsResult, err := GetTNS(schoolUUID, tenantUUID, *data.HomeroomTeacher)
		if err != nil {
			return fmt.Errorf("get homeroom teacher tns: %w", err)
		}
		if tnsResult == nil {
			return errors.New("homeroom teacher not found")
		}

		// insert homeroom teacher position
		_, err = tx.Exec(context.Background(), `
				insert into employee.employee_position 
				values
				($1, (select uuid from public."position" p where lower(p.abbr_name ) = 'wk' limit 1))
			`, tnsResult.EmployeeUUID)
		if err != nil {
			return fmt.Errorf("assign homeroom teacher role: %w", err)
		}

	}

	return tx.Commit(context.Background())
}

func CheckClassExists(data model.CreateClassModel, schoolUUID, tenantUUID uuid.UUID) (bool, error) {
	query := `select uuid, name, abbr_name, level, homeroom_teacher, status_uuid, created_date, updated_date, tenant_uuid
		from school_sch.class
		where tenant_uuid = $3 
		and (lower(name) = lower($1) or ($2::text is not null and lower(abbr_name) = lower($2))) 
		limit 1`
	selectedClass, err := db.GetSingleDataByQuery[model.ClassModel](query, data.Name, data.AbbrName, tenantUUID)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return false, err
		}
	}

	if selectedClass != nil {
		return true, nil
	}

	return false, nil
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
			count(st.uuid) as total_student
		from school_sch.class c
		left join user_sch.user u on c.homeroom_teacher = u.uuid
		left join public.status s on c.status_uuid = s.uuid
		left join school_sch.student st on c."uuid" = st.class_uuid 
		where c.tenant_uuid = $1
		group by c.uuid, c.name, c.abbr_name, c.level, u.name, s.name
	)
	`

	param = append(param, tenantUUID)
	queryBuilder := ""

	//* build query by payload data
	// search
	queryBuilder += `(lower(name) LIKE lower($` + strconv.Itoa(len(param)+1) + `) ` +
		` or lower(abbr_name) LIKE lower($` + strconv.Itoa(len(param)+1) + `) ` +
		` or lower(homeroom_teacher) LIKE lower($` + strconv.Itoa(len(param)+1) + `) ` +
		` or level::text LIKE lower($` + strconv.Itoa(len(param)+1) + `) ` +
		` or total_student::text LIKE lower($` + strconv.Itoa(len(param)+1) + `) ` +
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

	param = append(param, STATUS_DELETED)
	queryBuilder += " and lower(status) != lower($" + strconv.Itoa(len(param)) + ")"

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
