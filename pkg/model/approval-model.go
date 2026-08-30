package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MyApprovalResult struct {
	UUID          uuid.UUID `db:"uuid" json:"uuid"`
	WorkflowUUID  uuid.UUID `db:"workflow_uuid" json:"workflow_uuid"`
	WorkflowName  string    `db:"workflow_name" json:"workflow_name"`
	TicketNumber  string    `db:"ticket_number" json:"ticket_number"`
	CurrentStep   int       `db:"current_step" json:"current_step"`
	TotalStep     int       `db:"total_step" json:"total_step"`
	Status        string    `db:"status" json:"status"`
	RequestedBy   string    `db:"requested_by" json:"requested_by"`
	RoleName      string    `db:"role_name" json:"role_name"`
	RequestedDate time.Time `db:"requested_date" json:"requested_date"`
}

type DetailApprovalHeader struct {
	UUID          uuid.UUID       `db:"uuid" json:"uuid"`
	WorkflowUUID  uuid.UUID       `db:"workflow_uuid" json:"workflow_uuid"`
	WorkflowName  string          `db:"workflow_name" json:"workflow_name"`
	TicketNumber  string          `db:"ticket_number" json:"ticket_number"`
	CurrentStep   int             `db:"current_step" json:"current_step"`
	TotalStep     int             `db:"total_step" json:"total_step"`
	Status        string          `db:"status" json:"status"`
	RequestedBy   string          `db:"requested_by" json:"requested_by"`
	RoleName      string          `db:"role_name" json:"role_name"`
	RequestedDate time.Time       `db:"requested_date" json:"requested_date"`
	EntityType    string          `db:"entity_type" json:"entity_type"`
	EntityUUID    *string         `db:"entity_uuid" json:"entity_uuid"`
	RequestData   json.RawMessage `db:"request_data" json:"request_data"`
	ActionCode    string          `db:"action_code" json:"action_code"`
}

type DetailApprovalProgress struct {
	State       string     `db:"state" json:"state"`
	RoleName    string     `db:"role_name" json:"role_name"`
	ActBy       *string    `db:"act_by" json:"act_by"`
	ApproveDate *time.Time `db:"approve_date" json:"approve_date"`
	Note        *string    `db:"note" json:"note"`
}

type DetailApprovalModel struct {
	DetailApprovalHeader   DetailApprovalHeader     `json:"instance"`
	DetailApprovalProgress []DetailApprovalProgress `json:"progress"`
}
