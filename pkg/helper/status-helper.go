package helper

import (
	"gakuren-system.com/pkg/db"
	"gakuren-system.com/pkg/model"
)

func GetStatusByName(name string) (*model.StatusModel, error) {
	selectedStatus, err := db.GetSingleDataByQuery[model.StatusModel]("select * from public.status where lower(name) = lower($1)", name)
	if err != nil {
		return nil, err
	}

	return selectedStatus, nil
}
