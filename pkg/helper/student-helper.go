package helper

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GetTenantStudent(userUUID, tenantUUID uuid.UUID) (*model.UserModel, error) {
	data, err := db.GetSingleDataByQuery[model.UserModel](`
		select uuid, tenant_uuid, name, email, phone, address, img_location,
		       role_uuid, status_uuid, created_date, updated_date, version
		from user_sch."user" where uuid = $1 and tenant_uuid = $2
	`, userUUID, tenantUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	return data, err
}

func GetStudent(userUUID, tenantUUID uuid.UUID) (*model.StudentModel, error) {
	data, err := db.GetSingleDataByQuery[model.StudentModel](`
		select
			s."uuid"
			,s.user_uuid 
			,u."name"
			,s.nis 
			,s.nisn 
			,c.uuid class_uuid
			,u.phone 
			,u.email
			,g.uuid gender_uuid
			,s2.uuid status_uuid
			,u.address
			,s.parent_name 
			,s.parent_email 
			,s.parent_phone 
			,s.parent_address 
		from school_sch.student s 
		join user_sch."user" u on s.user_uuid = u."uuid" 
		left join school_sch."class" c on s.class_uuid = c."uuid" 
		join public.gender g on s.gender_uuid = g."uuid" 
		join public.status s2 on s.status_uuid = s2."uuid" 
		where s.uuid = $1 and u.tenant_uuid = $2
	`, userUUID, tenantUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	return data, err
}

func InsertUserStudent(data model.UserModel) (*uuid.UUID, error) {
	var id uuid.UUID
	err := db.Conn.QueryRow(context.Background(), `
		insert into user_sch."user"
			(tenant_uuid, name, email, phone, address, img_location, role_uuid,
			 status_uuid, created_date, updated_date, version, school_uuid)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning uuid
	`, data.TenantUUID, data.Name, data.Email, data.Phone, data.Address,
		data.ImgLocation, data.RoleUUID, data.StatusUUID, time.Now(),
		data.UpdatedDate, data.Version, data.SchoolUUID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &id, nil
}

func InsertStudent(data model.CreateStudentModel, userUUID, status uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := db.Conn.QueryRow(context.Background(), `
		INSERT INTO school_sch.student
		(user_uuid, gender_uuid, class_uuid, nis, nisn, status_uuid,
		parent_name,parent_phone,parent_email,parent_address, school_uuid)
		VALUES 
		($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning uuid
	`, userUUID, data.GenderUUID, data.ClassUUID, data.NIS, data.NISN, status,
		data.ParentName, data.ParentPhone, data.ParentEmail, data.ParentAddress, data.SchoolUUID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert student: %w", err)
	}
	return &id, nil
}

func UpdateStudent(data model.StudentModel) error {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	resultUpdateUser, err := tx.Exec(context.Background(), `
		update user_sch.user
			set name=$1, phone=$2, address=$3, updated_date=now()
		where uuid=$4
	`, data.Name, data.Phone, data.Address, data.UserUUID)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if resultUpdateUser.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	resultUpdateStudent, err := tx.Exec(context.Background(), `
		update school_sch.student
			set gender_uuid=$1, class_uuid=$2, nis=$3, nisn=$4, updated_date = now(),
			parent_name = $5, parent_email = $6, parent_phone = $7, parent_address = $8
		where uuid=$9
	`, data.GenderUUID, data.ClassUUID, data.NIS, data.NISN,
		data.ParentName, data.ParentEmail, data.ParentPhone, data.ParentAddress, data.UUID)
	if err != nil {
		return fmt.Errorf("update student: %w", err)
	}
	if resultUpdateStudent.RowsAffected() == 0 {
		return errors.New("student not found")
	}
	return tx.Commit(context.Background())
}

func UpdateStudentStatus(data model.StudentModel, tenantUUID, statusUUID uuid.UUID) error {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	result2, err := tx.Exec(context.Background(), `
		update school_sch.student set status_uuid=$1, updated_date=now() where uuid=$2
	`, statusUUID, data.UUID)
	if err != nil {
		return fmt.Errorf("update student status: %w", err)
	}
	if result2.RowsAffected() == 0 {
		return errors.New("student not found")
	}
	return tx.Commit(context.Background())
}

func UpdateStudentUserStatus(data model.StudentModel, tenantUUID, statusUUID uuid.UUID) error {
	tx, err := db.Conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	result1, err := tx.Exec(context.Background(), `
		update user_sch."user" set status_uuid=$1, updated_date=now() where uuid=$2 and tenant_uuid=$3
	`, statusUUID, data.UserUUID, tenantUUID)
	if err != nil {
		return fmt.Errorf("update user status: %w", err)
	}
	if result1.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return tx.Commit(context.Background())
}

func SoftDeleteStudent(data model.StudentModel, tenantUUID uuid.UUID) error {
	if err := UpdateStudentUserStatus(data, tenantUUID, DB_UUID_STATUS_INACTIVE); err != nil {
		return err
	}
	if err := UpdateStudentStatus(data, tenantUUID, DB_UUID_STUDENT_ENROLLMENT_INACTIVE); err != nil {
		return err
	}
	return nil
}

func SearchStudent(tenantUUID uuid.UUID, payload model.SearchPayload) ([]model.ReadStudentModelResult, *model.DataStatistics, error) {
	params := []interface{}{tenantUUID}
	base := `
	with datas as (
		select
			s."uuid"
			,s.user_uuid 
			,u."name"
			,s.nis 
			,s.nisn 
			,c."name" class_name
			,u.phone 
			,u.email
			,g."name" gender_name
			,s3.name as status
			,s2."name" student_status
			,u.address
			,s.parent_name 
			,s.parent_email 
			,s.parent_phone 
			,s.parent_address 
		from school_sch.student s 
		join user_sch."user" u on s.user_uuid = u."uuid" 
		left join school_sch."class" c on s.class_uuid = c."uuid" 
		join public.gender g on s.gender_uuid = g."uuid" 
		join public.status s2 on s.status_uuid = s2."uuid" 
		join public.status s3 on u.status_uuid = s3."uuid" 
		where u.tenant_uuid = $1
	)`
	search := ""
	if payload.Search != nil {
		search = strings.ToLower(strings.TrimSpace(*payload.Search))
	}
	params = append(params, "%"+search+"%")
	where := `(lower(coalesce(name,'')) like $2 or lower(coalesce(email,'')) like $2
		or lower(coalesce(phone,'')) like $2 or lower(coalesce(nis,'')) like $2
		or lower(coalesce(nisn,'')) like $2 )`
	if payload.Filter != nil {
		if status, ok := (*payload.Filter)["status"].(string); ok && status != "" {
			params = append(params, status)
			where += " and lower(student_status)=lower($" + strconv.Itoa(len(params)) + ")"
		}

		if id, ok := (*payload.Filter)["uuid"].(string); ok && id != "" {
			params = append(params, id)
			where += " and uuid=$" + strconv.Itoa(len(params))
		}
	}

	params = append(params, STATUS_DELETED)
	where += " and lower(student_status) != lower($" + strconv.Itoa(len(params)) + ")"

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
	rows, err := db.GetMultipleDataByQuery[model.ReadStudentModelResult](query, params...)
	if err != nil {
		return nil, nil, err
	}

	stats := CalculateDataStatisticResult(count, payload, len(*rows))
	return *rows, &stats, nil
}

func UserStudentValidity(data model.CreateStudentModel) error {
	checkUser, err := db.GetMultipleDataByQuery[model.UserModel](`
	with datas as (
		select 
			* 
		from user_sch.user u
		join school_sch.student s on u.uuid = s.user_uuid 
	)
	select email from datas
	where 
		lower(email) like lower($1) or 
		lower(name) like lower($2) or 
		lower(nis) like lower($3) or 
		lower(nisn) like lower($4) or 
		lower(phone) like lower($5)
	`, "%"+*data.Email+"%", "%"+*data.Name+"%", "%"+*data.NIS+"%", "%"+*data.NISN+"%", "%"+*data.Phone+"%")

	if err != nil {
		return err
	}
	if len(*checkUser) > 0 {
		return errors.New("Multiple user found")
	}
	return nil
}

func GetUserStatus(statusCode string) (*uuid.UUID, error) {
	var statusUUID uuid.UUID
	err := db.Conn.QueryRow(context.Background(), `select uuid from public.status where lower(category) = 'student_enrollment' and lower(code) = lower($1)`, statusCode).Scan(&statusUUID)
	if err != nil {
		return nil, err
	}
	return &statusUUID, nil
}
