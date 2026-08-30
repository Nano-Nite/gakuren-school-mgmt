package model

import (
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
