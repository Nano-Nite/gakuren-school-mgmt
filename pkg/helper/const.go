package helper

import "os"

var API_VERSION = os.Getenv("API_VERSION")

//DEFAULT SETTING
const DEFAULT_ROW_PER_PAGES = 10
const DEFAULT_PAGES = 1

// STATUS NAME
const STATUS_ACTIVE = "Active"
const STATUS_INACTIVE = "Inactive"

func init() {
	if API_VERSION == "" {
		API_VERSION = "v1"
	}
}
