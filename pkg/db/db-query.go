package db

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GetSingleDataByQuery[T any](query string, param ...interface{}) (*T, error) {
	rows, err := Conn.Query(context.Background(), query, param...)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	result, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[T])
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return &result, nil
}

func GetMultipleDataByQuery[T any](query string, param ...interface{}) (*[]T, error) {
	rows, err := Conn.Query(context.Background(), query, param...)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	result, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[T])
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return &result, nil
}

func ExecuteQuery(ctx context.Context, query string, param ...interface{}) error {
	_, err := Conn.Exec(ctx, query, param...)
	if err != nil {
		if err.Error() != "no rows in result set" {
			log.Println(err.Error())
			return err
		}
	}

	return nil
}

func InsertReturnUUID(query string, param ...interface{}) (*uuid.UUID, error) {
	var id uuid.UUID
	err := Conn.QueryRow(context.Background(), query, param...).Scan(&id)
	if err != nil {
		if err.Error() != "no rows in result set" {
			log.Println(err.Error())
			return nil, err
		}
	}

	return &id, nil
}
