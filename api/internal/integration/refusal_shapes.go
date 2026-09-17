package integration

type PublicationError struct {
	Code  PublicationErrorCode `json:"code"`
	Error string               `json:"error"`
	Field *string              `json:"field,omitempty"`
}

type PublicationErrorCode string

const (
	PublicationErrorCodeAlreadyScheduled PublicationErrorCode = "already_scheduled"
	PublicationErrorCodeCategoryRefused  PublicationErrorCode = "category_refused"
	PublicationErrorCodeForbidden        PublicationErrorCode = "forbidden"
	PublicationErrorCodeInvalid          PublicationErrorCode = "invalid"
	PublicationErrorCodeNotFound         PublicationErrorCode = "not_found"
	PublicationErrorCodeScheduleRunning  PublicationErrorCode = "schedule_running"
	PublicationErrorCodeServerError      PublicationErrorCode = "server_error"
	PublicationErrorCodeStaleVersion     PublicationErrorCode = "stale_version"
	PublicationErrorCodeUnauthenticated  PublicationErrorCode = "unauthenticated"
)
