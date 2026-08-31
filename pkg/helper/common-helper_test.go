package helper

import (
	"testing"
	"time"

	"gakuren-system.com/pkg/model"
	"github.com/google/uuid"
)

func TestMapIntoStuctConvertsClassModelValues(t *testing.T) {
	statusUUID := uuid.MustParse("681a8dac-4ced-4de3-9254-0c42e6bdc38b")
	tenantUUID := uuid.MustParse("ae1368b8-bec5-4a0a-9c4f-dae79a1d5beb")
	createdDate := "2026-08-30T23:03:08.877880512Z"

	got, err := MapIntoStuct[model.ClassModel](map[string]interface{}{
		"UUID":            nil,
		"AbbrName":        "XIIRPL1",
		"CreatedDate":     createdDate,
		"UpdatedDate":     nil,
		"HomeroomTeacher": nil,
		"Name":            "XII RPL 1",
		"Level":           float64(12),
		"StatusUUID":      statusUUID.String(),
		"TenantUUID":      tenantUUID.String(),
	})
	if err != nil {
		t.Fatalf("MapIntoStuct returned an error: %v", err)
	}

	wantCreatedDate, _ := time.Parse(time.RFC3339Nano, createdDate)
	if got.StatusUUID != statusUUID || got.TenantUUID != tenantUUID {
		t.Fatalf("UUID fields were not converted: status=%s tenant=%s", got.StatusUUID, got.TenantUUID)
	}
	if !got.CreatedDate.Equal(wantCreatedDate) {
		t.Fatalf("CreatedDate = %s, want %s", got.CreatedDate, wantCreatedDate)
	}
	if got.UUID != nil || got.UpdatedDate != nil || got.HomeroomTeacher != nil {
		t.Fatal("nullable fields should remain nil")
	}
	if got.AbbrName == nil || *got.AbbrName != "XIIRPL1" || got.Level != 12 {
		t.Fatalf("ordinary fields were not converted correctly: %+v", got)
	}
}
