package blog

import "strings"

type FieldError struct {
	Field   string
	Message string
	Cause   error
}

func (e FieldError) Error() string { return e.Message }

func (e FieldError) Unwrap() error { return e.Cause }

func oneParagraph(written string) string {
	return strings.Join(strings.Fields(written), " ")
}
