package helper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
	"github.com/google/uuid"
)

// func GetTenantStudent(userUUID, tenantUUID uuid.UUID) (*model.UserModel, error) {
// 	data, err := db.GetSingleDataByQuery[model.UserModel](`
// 		select uuid, tenant_uuid, name, email, phone, address, img_location,
// 		       role_uuid, status_uuid, created_date, updated_date, version
// 		from user_sch."user" where uuid = $1 and tenant_uuid = $2
// 	`, userUUID, tenantUUID)
// 	if errors.Is(err, pgx.ErrNoRows) {
// 		return nil, errors.New("user not found")
// 	}
// 	return data, err
// }

// func GetStudent(userUUID, tenantUUID uuid.UUID) (*model.StudentModel, error) {
// 	data, err := db.GetSingleDataByQuery[model.StudentModel](`
// 		select
// 			s."uuid"
// 			,s.user_uuid
// 			,u."name"
// 			,s.nis
// 			,s.nisn
// 			,c.uuid class_uuid
// 			,u.phone
// 			,u.email
// 			,g.uuid gender_uuid
// 			,s2.uuid status_uuid
// 			,u.address
// 			,s.parent_name
// 			,s.parent_email
// 			,s.parent_phone
// 			,s.parent_address
// 		from school_sch.student s
// 		join user_sch."user" u on s.user_uuid = u."uuid"
// 		left join school_sch."class" c on s.class_uuid = c."uuid"
// 		join public.gender g on s.gender_uuid = g."uuid"
// 		join public.status s2 on s.status_uuid = s2."uuid"
// 		where s.uuid = $1 and u.tenant_uuid = $2
// 	`, userUUID, tenantUUID)
// 	if errors.Is(err, pgx.ErrNoRows) {
// 		return nil, errors.New("user not found")
// 	}
// 	return data, err
// }

func InsertTNS(data model.CreateTNSModel, userData model.UserModel, status, tenantUUID, schoolUUID uuid.UUID) (*uuid.UUID, error) {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	var userUUID uuid.UUID
	var employeeUUID uuid.UUID

	// create user
	err = tx.QueryRow(context.Background(), `
		insert into user_sch."user"
			(tenant_uuid, name, email, phone, address, img_location, role_uuid,
			 status_uuid, created_date, updated_date, version, school_uuid)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning uuid
	`,
		userData.TenantUUID,
		userData.Name,
		userData.Email,
		userData.Phone,
		userData.Address,
		userData.ImgLocation,
		userData.RoleUUID,
		userData.StatusUUID,
		userData.CreatedDate,
		userData.UpdatedDate,
		userData.Version,
		schoolUUID).Scan(&userUUID)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	// insert employee
	err = tx.QueryRow(context.Background(), `
		INSERT INTO employee.employee
			(user_uuid, nik, nuptk, nip, gender_uuid, join_date, status_uuid, birth_place, birth_date)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9)
		returning uuid
	`,
		userUUID,
		data.NIK,
		data.NUPTK,
		data.NIP,
		data.Biodata.GenderUUID,
		data.JoinDate,
		status,
		data.Biodata.BirthPlace,
		data.Biodata.BirthDate).Scan(&employeeUUID)
	if err != nil {
		return nil, fmt.Errorf("insert employee: %w", err)
	}

	// insert education
	if data.EducationLevel != nil {
		var idEdu string
		for _, edu := range data.EducationLevel {
			err = tx.QueryRow(context.Background(), `
			INSERT INTO employee.employee_education
				(employee_uuid, education_level_uuid, institution_name, major, start_year, end_year, is_latest)
			VALUES
				($1, $2, $3, $4, $5, $6, $7)
			RETURNING uuid
		`,
				employeeUUID,
				edu.EducationLevelUUID,
				edu.InstitutionName,
				edu.Major,
				edu.StartYear,
				edu.EndYear,
				edu.LastEducation,
			).Scan(&idEdu)
			if err != nil {
				return nil, fmt.Errorf("insert employee education: %w", err)
			}
		}
	}

	// insert position
	if data.Positions != nil {
		var positions uuid.UUIDs

		// validate and get positions uuid
		for _, pos := range data.Positions {
			var idPos uuid.UUID
			err = tx.QueryRow(context.Background(), `
			select p.uuid
			from public.position p
			where p.uuid = $1
		`,
				pos.UUID,
			).Scan(&idPos)
			if err != nil {
				return nil, fmt.Errorf("insert employee position: %w", err)
			}
			positions = append(positions, idPos)
		}

		// insert employee_position
		for _, v := range positions {
			var insertedIDPos uuid.UUID
			err = tx.QueryRow(context.Background(), `
			insert into employee.employee_position
				(employee_uuid, position_uuid)
			values
				($1, $2)
			returning employee_uuid
		`,
				employeeUUID,
				v,
			).Scan(&insertedIDPos)
			if err != nil {
				return nil, fmt.Errorf("insert employee position: %w", err)
			}
		}
	}

	// insert title
	if data.Titles != nil {
		var titles uuid.UUIDs

		// validate and get title uuid
		for _, title := range data.Titles {
			var idtitle uuid.UUID
			err = tx.QueryRow(context.Background(), `
			select t.uuid
			from public.title t
			where t.uuid = $1
		`,
				title.UUID,
			).Scan(&idtitle)
			if err != nil {
				return nil, fmt.Errorf("insert employee position: %w", err)
			}
			titles = append(titles, idtitle)
		}

		// insert employee_title
		for _, v := range titles {
			var insertedIDtitle uuid.UUID
			err = tx.QueryRow(context.Background(), `
			insert into employee.employee_title
				(employee_uuid, title_uuid)
			values
				($1, $2)
			returning employee_uuid
		`,
				employeeUUID,
				v,
			).Scan(&insertedIDtitle)
			if err != nil {
				return nil, fmt.Errorf("insert employee title: %w", err)
			}
		}
	}

	// insert subject
	if data.Subjects != nil {
		var subjects uuid.UUIDs

		// validate and get subject uuid
		for _, subject := range data.Subjects {
			var idSubject uuid.UUID
			err = tx.QueryRow(context.Background(), `
			select 
				uuid
			from public.subject
			where uuid = $1 and school_uuid = $2 and tenant_uuid = $3
		`,
				subject.UUID,
				schoolUUID,
				tenantUUID,
			).Scan(&idSubject)
			if err != nil {
				return nil, fmt.Errorf("insert employee subject: %w", err)
			}
			subjects = append(subjects, idSubject)
		}

		// insert employee_subject
		for _, v := range subjects {
			var insertedIDsubject uuid.UUID
			err = tx.QueryRow(context.Background(), `
			insert into employee.employee_subject
				(employee_uuid, subject_uuid)
			values
				($1, $2)
			returning employee_uuid
		`,
				employeeUUID,
				v,
			).Scan(&insertedIDsubject)
			if err != nil {
				return nil, fmt.Errorf("insert employee subject: %w", err)
			}
		}
	}

	return &employeeUUID, tx.Commit(context.Background())
}

// func UpdateStudent(data model.StudentModel) error {
// 	tx, err := db.Conn.Begin(context.Background())
// 	if err != nil {
// 		return err
// 	}
// 	defer tx.Rollback(context.Background())

// 	resultUpdateUser, err := tx.Exec(context.Background(), `
// 		update user_sch.user
// 			set name=$1, email=$2, phone=$3, address=$4, updated_date=now()
// 		where uuid=$5
// 	`, data.Name, data.Email, data.Phone, data.Address, data.UserUUID)
// 	if err != nil {
// 		return fmt.Errorf("update user: %w", err)
// 	}
// 	if resultUpdateUser.RowsAffected() == 0 {
// 		return errors.New("user not found")
// 	}
// 	resultUpdateStudent, err := tx.Exec(context.Background(), `
// 		update school_sch.student
// 			set gender_uuid=$1, class_uuid=$2, nis=$3, nisn=$4, status_uuid=$5, updated_date = now(),
// 			parent_name = $6, parent_email = $7, parent_phone = $8, parent_address = $9
// 		where uuid=$10
// 	`, data.GenderUUID, data.ClassUUID, data.NIS, data.NISN, data.StatusUUID,
// 		data.ParentName, data.ParentEmail, data.ParentPhone, data.ParentAddress, data.UUID)
// 	if err != nil {
// 		return fmt.Errorf("update user: %w", err)
// 	}
// 	if resultUpdateStudent.RowsAffected() == 0 {
// 		return errors.New("user not found")
// 	}
// 	return tx.Commit(context.Background())
// }

// func UpdateStudentStatus(data model.StudentModel, tenantUUID, statusUUID uuid.UUID) error {
// 	tx, err := db.Conn.Begin(context.Background())
// 	if err != nil {
// 		return err
// 	}
// 	defer tx.Rollback(context.Background())

// 	//! we will think about it later, do we need to make user disabled too or not.
// 	//! when user inactive during pending student status, they probably cannot login
// 	// result1, err := tx.Exec(context.Background(), `
// 	// 	update user_sch."user" set status_uuid=$1, updated_date=now() where uuid=$2 and tenant_uuid=$3
// 	// `, statusUUID, data.UserUUID, tenantUUID)
// 	// if err != nil {
// 	// 	return fmt.Errorf("update user status: %w", err)
// 	// }
// 	// if result1.RowsAffected() == 0 {
// 	// 	return errors.New("user not found")
// 	// }

// 	result2, err := tx.Exec(context.Background(), `
// 		update school_sch.student set status_uuid=$1, updated_date=now() where uuid=$2
// 	`, statusUUID, data.UUID)
// 	if err != nil {
// 		return fmt.Errorf("update user status: %w", err)
// 	}
// 	if result2.RowsAffected() == 0 {
// 		return errors.New("user not found")
// 	}
// 	return tx.Commit(context.Background())
// }

// func SoftDeleteStudent(data model.StudentModel, tenantUUID uuid.UUID) error {
// 	return UpdateStudentStatus(data, tenantUUID, DB_UUID_STATUS_INACTIVE)
// }

func SearchTNSHeader(schoolUUID, tenantUUID, userUUID uuid.UUID, payload model.SearchPayload) ([]model.ReadTNSHeaderModelResult, *model.DataStatistics, error) {
	params := []interface{}{tenantUUID, schoolUUID}
	base := `
	with datas as (
		select
			s.uuid
			,s.name
			,e.nip 
			,s.email 
			,s.phone 
			,r.name as occupation
			,s2.name as status
		from user_sch.user s
		join employee.employee e on s.uuid = e.user_uuid  
		join public.status s2 on e.status_uuid = s2.uuid
		join user_sch.role r on s.role_uuid = r.uuid
		where s.tenant_uuid = $1 and s.school_uuid = $2

	)`

	search := ""
	if payload.Search != nil {
		search = strings.ToLower(strings.TrimSpace(*payload.Search))
	}
	params = append(params, "%"+search+"%")
	where := `(
		lower(coalesce(name,'')) like $3 
		or lower(coalesce(email,'')) like $3
		or lower(coalesce(phone,'')) like $3 
		or lower(coalesce(nip,'')) like $3
	)`
	if payload.Filter != nil {
		if status, ok := (*payload.Filter)["status"].(string); ok && status != "" {
			params = append(params, status)
			where += " and lower(status)=lower($" + strconv.Itoa(len(params)) + ")"
		}

		if id, ok := (*payload.Filter)["uuid"].(string); ok && id != "" {
			params = append(params, id)
			where += " and uuid=$" + strconv.Itoa(len(params))
		}
	}

	params = append(params, STATUS_DELETE)
	where += " and lower(status) != lower($" + strconv.Itoa(len(params)) + ")"

	log.Println(base + " select count(*) from datas where " + where)

	count, err := db.GetSingleDataByQuery[model.CountResult](base+" select count(*) from datas where "+where, params...)
	if err != nil {
		return nil, nil, err
	}
	order := " order by name asc"
	allowed := map[string]bool{"name": true, "email": true, "phone": true, "role_name": true, "status": true, "version": true}
	if payload.SortBy != nil {
		for _, item := range *payload.SortBy {
			for column, rawDirection := range item {
				direction, ok := rawDirection.(string)
				if allowed[column] && ok && (strings.EqualFold(direction, "asc") || strings.EqualFold(direction, "desc")) {
					order = " order by " + column + " " + direction
					break
				}
			}
		}
	}
	limit, page := DEFAULT_ROW_PER_PAGES, DEFAULT_PAGES
	if payload.RowPerPage != nil && *payload.RowPerPage > 0 {
		limit = *payload.RowPerPage
	}
	if payload.Page != nil && *payload.Page > 0 {
		page = *payload.Page
	}
	params = append(params, limit, (page-1)*limit)
	query := base + " select * from datas where " + where + order +
		" limit $" + strconv.Itoa(len(params)-1) + " offset $" + strconv.Itoa(len(params))
	rows, err := db.GetMultipleDataByQuery[model.ReadTNSHeaderModelResult](query, params...)
	if err != nil {
		return nil, nil, err
	}

	stats := CalculateDataStatisticResult(count, payload, len(*rows))
	return *rows, &stats, nil
}

func SearchTNSDetail(schoolUUID, tenantUUID, userUUID, tnsUUID uuid.UUID) (*model.ReadTNSDetailModelResult, error) {
	query := `
	select
		s.uuid
		,s.name 
		,s.email 
		,s.phone
		,s.address 
		,e.nik
		,e.nuptk 
		,e.nip
		,g.name gender
		,e.birth_place 
		,e.birth_date::date 
		,e.join_date::date
		,e.resign_date::date
		,(
			SELECT COALESCE(
				jsonb_agg(
					jsonb_build_object(
						'institution_name', ee.institution_name, 
						'code', el.code, 
						'major', ee.major, 
						'start_year', ee.start_year, 
						'end_year', ee.end_year, 
						'is_latest', ee.is_latest
				) order by el.level_order desc
			), '[]'::jsonb) 
			FROM employee.employee_education ee
			join public.education_level el on ee.education_level_uuid = el.uuid 
			WHERE ee.employee_uuid = e.uuid
		) AS educations
		,(
			SELECT COALESCE(jsonb_agg(p.name), '[]'::jsonb)
			FROM employee.employee_position ep 
			JOIN public.position p ON p.uuid = ep.position_uuid  
			WHERE ep.employee_uuid = e.uuid
		) AS positions
		,(
			select coalesce(
				jsonb_agg(
					jsonb_build_object(
						'abbr_name', t.abbr_name, 
						'is_prefix', t.is_prefix, 
						'sequence', t.sequence
					)
				), '[]'::jsonb)
			from employee.employee_title et 
			join public.title t on et.title_uuid = t.uuid
			where et.employee_uuid = e.uuid
		) as titles
		,(
			select coalesce(jsonb_agg(s3.name), '[]'::jsonb)
			from employee.employee_subject es 
			join public.subject s3 on es.subject_uuid = s3.uuid
			where es.employee_uuid = e.uuid
		) as subject
		,s2.name as status
	from user_sch.user s
	join employee.employee e on s.uuid = e.user_uuid  
	join public.status s2 on e.status_uuid = s2.uuid
	join public.gender g on e.gender_uuid = g.uuid
	where s.tenant_uuid = $1 
	and s.school_uuid = $2
	and s.uuid = $3
	`
	selectedDetail, err := db.GetSingleDataByQuery[model.ReadTNSDetailModelResult](query, tenantUUID, schoolUUID, tnsUUID)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return nil, err
		}
	}
	if selectedDetail == nil {
		return nil, errors.New("Fail to get detail data")
	}

	return selectedDetail, nil
}

func UserTNSValidity(data model.CreateTNSModel) error {
	query := `
	select * from user_sch.user
	where
		lower(email) like lower($1) or
		lower(name) like lower($2) or
		lower(phone) like lower($3)
	`

	checkUser, err := db.GetMultipleDataByQuery[model.UserModel](query,
		fmt.Sprintf("%%%s%%", strings.ToLower(data.Biodata.Email)),
		fmt.Sprintf("%%%s%%", strings.ToLower(data.Biodata.Fullname)),
		fmt.Sprintf("%%%s%%", strings.Trim(data.Biodata.Phone, "")),
	)
	if err != nil {
		return err
	}
	if len(*checkUser) > 0 {
		return errors.New("Multiple user found")
	}

	return nil
}
