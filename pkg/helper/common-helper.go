package helper

import (
	"encoding/base64"
	"errors"
	"log"
	"math"
	"strings"

	"gakuren-system.com/pkg/model"
	"github.com/gofiber/fiber/v3"
)

func ReturnResponse(c fiber.Ctx, statusCode int, message string, data interface{}, err error) error {
	response := make(map[string]interface{})
	response["message"] = message
	response["data"] = nil
	response["error"] = nil

	if data != nil {
		response["data"] = data
	}
	if err != nil {
		log.Println(err)
		response["error"] = err.Error()
	}
	return c.Status(statusCode).JSON(response)
}

func DecodeB64String(src string) (string, error) {
	decodedByte, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return "", err
	}

	return string(decodedByte), nil
}

func DecodeB64Bytes(src string) ([]byte, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil, errors.New("empty base64 input")
	}

	b, err := base64.StdEncoding.DecodeString(src)
	if err == nil {
		return b, nil
	}

	b, err = base64.URLEncoding.DecodeString(src)
	if err == nil {
		return b, nil
	}

	return nil, err
}

func CalculateDataStatisticResult(count *model.CountResult, payload model.SearchPayload, totalData int) model.DataStatistics {
	var dataStat model.DataStatistics
	dataStat.TotalRow = count.Count
	if payload.RowPerPage != nil {
		dataStat.RowPerPage = *payload.RowPerPage
	} else {
		dataStat.RowPerPage = DEFAULT_ROW_PER_PAGES
	}
	if payload.Page != nil {
		dataStat.CurrentPage = *payload.Page
	} else {
		dataStat.CurrentPage = DEFAULT_PAGES
	}
	dataStat.MaxPage = int(math.Ceil(float64(count.Count) / float64(dataStat.RowPerPage)))
	dataStat.CurrentRow = totalData
	if (dataStat.CurrentPage * dataStat.RowPerPage) > dataStat.TotalRow {
		dataStat.StartRow = dataStat.TotalRow - dataStat.CurrentRow + 1
	} else {
		dataStat.StartRow = (dataStat.CurrentPage * dataStat.RowPerPage) - (dataStat.RowPerPage - 1)
	}
	if dataStat.CurrentPage*dataStat.RowPerPage < dataStat.TotalRow {
		dataStat.EndRow = dataStat.CurrentPage * dataStat.RowPerPage
	} else {
		dataStat.EndRow = dataStat.TotalRow
	}

	return dataStat
}
