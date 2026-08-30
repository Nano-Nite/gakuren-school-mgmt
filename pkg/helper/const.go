package helper

import "os"

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

// ACTION CODE
const ACTION_CODE_CREATE = "CREATE"
const ACTION_CODE_SUBMIT = "SUBMIT"

// COMMON PERMISSION
const APPROVAL_BYPASS = "appr.bypass"

// CLASS PERMISSION
const CREATE_CLASS_PERMISSION = "class.create"
const UPDATE_CLASS_PERMISSION = "class.update"
const DELETE_CLASS_PERMISSION = "class.delete"

// const SUBMIT_CLASS_PERMISSION = "class.submit"
// const APPROVE_CLASS_PERMISSION = "class.approve"
// const REJECT_CLASS_PERMISSION = "class.reject"

func init() {
	if API_VERSION == "" {
		API_VERSION = "v1"
	}
}
