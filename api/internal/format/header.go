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

// FileFields names the page fields a type's downloaded file carries, so editing them waits for Publish.
func (r *Registry) FileFields(workType string) []string {
	fields := make([]string, 0)
	for _, module := range r.modules {
		declaration := module.Declaration()
		if !declaration.Direction.Write || declaration.Type != workType {
			continue
		}
		if declaration.WritesCover {
			fields = append(fields, "cover")
		}
		for _, field := range declaration.Header {
			if field == HeaderName || field == HeaderBlurb {
				fields = append(fields, string(field))
			}
		}
	}
	slices.Sort(fields)
	return slices.Compact(fields)
}
