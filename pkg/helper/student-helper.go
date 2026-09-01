package helper

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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

func InsertStudent(data model.UserModel) (*uuid.UUID, error) {
	var id uuid.UUID
	err := db.Conn.QueryRow(db.DBCtx, `
		insert into user_sch."user"
			(tenant_uuid, name, email, phone, address, img_location, role_uuid,
			 status_uuid, created_date, updated_date, version)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning uuid
	`, data.TenantUUID, data.Name, data.Email, data.Phone, data.Address,
		data.ImgLocation, data.RoleUUID, data.StatusUUID, data.CreatedDate,
		data.UpdatedDate, data.Version).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &id, nil
}

func UpdateStudent(data model.UserModel) error {
	result, err := db.Conn.Exec(db.DBCtx, `
		update user_sch."user"
		set name=$1, email=$2, phone=$3, address=$4, img_location=$5,
		    role_uuid=$6, version=$7, updated_date=now()
		where uuid=$8 and tenant_uuid=$9
	`, data.Name, data.Email, data.Phone, data.Address, data.ImgLocation,
		data.RoleUUID, data.Version, data.UUID, data.TenantUUID)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func UpdateStudentStatus(userUUID, tenantUUID, statusUUID uuid.UUID) error {
	result, err := db.Conn.Exec(db.DBCtx, `
		update user_sch."user" set status_uuid=$1, updated_date=now()
		where uuid=$2 and tenant_uuid=$3
	`, statusUUID, userUUID, tenantUUID)
	if err != nil {
		return fmt.Errorf("update user status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func SoftDeleteStudent(userUUID, tenantUUID uuid.UUID) error {
	return UpdateStudentStatus(userUUID, tenantUUID, DB_UUID_STATUS_INACTIVE)
}

func SearchStudent(tenantUUID uuid.UUID, payload model.SearchPayload) ([]model.ReadStudentModelResult, *model.DataStatistics, error) {
	params := []interface{}{tenantUUID}
	base := `with datas as (
		select u.uuid,u.name,u.email,u.phone,u.address,u.img_location,u.role_uuid,
		       r.name role_name,s.name status,u.version
		from user_sch."user" u
		join user_sch.role r on r.uuid=u.role_uuid
		join public.status s on s.uuid=u.status_uuid
		where u.tenant_uuid=$1
	)`
	search := ""
	if payload.Search != nil {
		search = strings.ToLower(strings.TrimSpace(*payload.Search))
	}
	params = append(params, "%"+search+"%")
	where := `(lower(coalesce(name,'')) like $2 or lower(coalesce(email,'')) like $2
		or lower(coalesce(phone,'')) like $2 or lower(coalesce(role_name,'')) like $2)`
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
