package format

import "slices"

type HeaderField string

const (
	HeaderName           HeaderField = "name"
	HeaderBlurb          HeaderField = "blurb"
	HeaderWorkVersion    HeaderField = "work_version"
	HeaderCreditedAuthor HeaderField = "credited_author"
	HeaderNickname       HeaderField = "nickname"
)

func HeaderFields() []HeaderField {
	return []HeaderField{
		HeaderName, HeaderBlurb, HeaderWorkVersion, HeaderCreditedAuthor,
		HeaderNickname,
	}
}

func (f HeaderField) Known() bool { return slices.Contains(HeaderFields(), f) }

func (r *Registry) ExportedHeaderFields(workType string) []HeaderField {
	fields := make([]HeaderField, 0)
	for _, module := range r.modules {
		declaration := module.Declaration()
		if !declaration.Direction.Write || declaration.Type != workType {
			continue
		}
		for _, field := range declaration.Header {
			if !slices.Contains(fields, field) {
				fields = append(fields, field)
			}
		}
	}
	slices.Sort(fields)
	return fields
}
