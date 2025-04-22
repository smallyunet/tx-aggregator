package model

import (
	"log"
)

// Error codes used throughout the application
const (
	CodeSuccess        = 0    // Operation completed successfully
	CodeInvalidParam   = 1001 // Invalid input parameters
	CodeInternalError  = 1002 // Internal server error
	CodeProviderFailed = 1003 // Failed to get data from external provider
	CodeTimeout        = 1004 // Request timed out
)

// CodeMessageMap maps error codes to their corresponding error messages
var CodeMessageMap = map[int]string{
	CodeSuccess:        "success",
	CodeInvalidParam:   "invalid parameters",
	CodeInternalError:  "internal server error",
	CodeProviderFailed: "failed to get transactions from provider",
}

// GetMessageByCode returns the error message for a given error code.
// If the code is not found in the map, it returns "unknown error".
func GetMessageByCode(code int) string {
	if msg, ok := CodeMessageMap[code]; ok {
		return msg
	}
	// Log warning for unknown error code
	log.Printf("WARNING: Unknown error code encountered: %d", code)
	return "unknown error"
}
