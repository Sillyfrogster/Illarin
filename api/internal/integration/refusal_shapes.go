package integration

type BlogError struct {
	Code  BlogErrorCode `json:"code"`
	Error string        `json:"error"`
	Field *string       `json:"field,omitempty"`
}

type BlogErrorCode string

const (
	BlogErrorCodeAlreadyScheduled BlogErrorCode = "already_scheduled"
	BlogErrorCodeCategoryRefused  BlogErrorCode = "category_refused"
	BlogErrorCodeForbidden        BlogErrorCode = "forbidden"
	BlogErrorCodeInvalid          BlogErrorCode = "invalid"
	BlogErrorCodeNotFound         BlogErrorCode = "not_found"
	BlogErrorCodeScheduleRunning  BlogErrorCode = "schedule_running"
	BlogErrorCodeServerError      BlogErrorCode = "server_error"
	BlogErrorCodeStaleVersion     BlogErrorCode = "stale_version"
	BlogErrorCodeUnauthenticated  BlogErrorCode = "unauthenticated"
)
