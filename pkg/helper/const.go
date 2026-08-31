package helper

import (
	"log"
	"os"

	"github.com/google/uuid"
)

var API_VERSION = os.Getenv("API_VERSION")

// COMMON CONST
const REJECT = "REJECT"
const SAVE = "SAVE"
const WORKFLOW_NOTFOUND_BEHAVIOUR = "workflow.notfound.behaviour"
const APPROVAL_DOCUMENT_CODE = "APR"

// ENTITY TYPE
const CLASS_ENTITY_TYPE = "CLASS"

// MODULE CODE
const CLASS_MODULE_CODE = "KLS"

// DEFAULT SETTING
const DEFAULT_ROW_PER_PAGES = 10
const DEFAULT_PAGES = 1

// STATUS NAME
const STATUS_ACTIVE = "Active"
const STATUS_INACTIVE = "Inactive"
const STATUS_PENDING = "Pending"

// ACTION CODE
const ACTION_CODE_CREATE = "CREATE"
const ACTION_CODE_UPDATE = "UPDATE"
const ACTION_CODE_DELETE = "DELETE"
const ACTION_CODE_SUBMIT = "SUBMIT"
const ACTION_CODE_CANCEL = "CANCEL"
const ACTION_CODE_REJECT = "REJECT"
const ACTION_CODE_APPROVE = "APPROVE"

// COMMON PERMISSION
const APPROVAL_BYPASS = "appr.bypass"

// CLASS PERMISSION
const CREATE_CLASS_PERMISSION = "class.create"
const UPDATE_CLASS_PERMISSION = "class.update"
const DELETE_CLASS_PERMISSION = "class.delete"

// const SUBMIT_CLASS_PERMISSION = "class.submit"
// const APPROVE_CLASS_PERMISSION = "class.approve"
// const REJECT_CLASS_PERMISSION = "class.reject"

// DB STATUS VAR
var DB_UUID_STATUS_ACTIVE uuid.UUID
var DB_UUID_STATUS_PENDING uuid.UUID
var DB_UUID_STATUS_INACTIVE uuid.UUID

func init() {
	if API_VERSION == "" {
		API_VERSION = "v1"
	}
}

func InitVariableDB() {
	activeStatus, err := GetStatusByName(STATUS_ACTIVE)
	if err != nil {
		log.Fatal("Critical error loading configuration: ", err)
	} else {
		DB_UUID_STATUS_ACTIVE = activeStatus.UUID
	}
	pendingStatus, err := GetStatusByName(STATUS_PENDING)
	if err != nil {
		log.Fatal("Critical error loading configuration: ", err)
	} else {
		DB_UUID_STATUS_PENDING = pendingStatus.UUID
	}
	inactiveStatus, err := GetStatusByName(STATUS_INACTIVE)
	if err != nil {
		log.Fatal("Critical error loading configuration: ", err)
	} else {
		DB_UUID_STATUS_INACTIVE = inactiveStatus.UUID
	}

	log.Println("Variable Loaded")
}
