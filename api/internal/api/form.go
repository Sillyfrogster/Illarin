package api

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
)

const (
	MetadataPart = "metadata"
	FilePart     = "file"
)

// FormRefusal says what was wrong with a form a person sent
type FormRefusal struct {
	Reason string
	Cause  error
}

func (r FormRefusal) Error() string { return r.Reason }
func (r FormRefusal) Unwrap() error { return r.Cause }

// NextPart reads the next part of a form and refuses it when it is not the named one
func NextPart(parts *multipart.Reader, name string) (*multipart.Part, error) {
	part, err := parts.NextPart()
	if errors.Is(err, io.EOF) {
		return nil, FormRefusal{Reason: "the " + name + " part is missing", Cause: err}
	}
	if err != nil {
		return nil, FormRefusal{Reason: "the form data could not be read", Cause: err}
	}
	if part.FormName() != name {
		return nil, FormRefusal{
			Reason: fmt.Sprintf("expected the %s part here, found %q", name, part.FormName()),
			Cause:  nil,
		}
	}
	return part, nil
}
