package http

type PublicationError struct {
	Code  PublicationErrorCode `json:"code"`
	Error string               `json:"error"`
	Field *string              `json:"field,omitempty"`
}

type PublicationErrorCode string

const (
	CodeAlreadyScheduled PublicationErrorCode = "already_scheduled"
	CodeCategoryRefused  PublicationErrorCode = "category_refused"
	CodeForbidden        PublicationErrorCode = "forbidden"
	CodeInvalid          PublicationErrorCode = "invalid"
	CodeNotFound         PublicationErrorCode = "not_found"
	CodeScheduleRunning  PublicationErrorCode = "schedule_running"
	CodeServerError      PublicationErrorCode = "server_error"
	CodeStaleVersion     PublicationErrorCode = "stale_version"
	CodeUnauthenticated  PublicationErrorCode = "unauthenticated"
)
