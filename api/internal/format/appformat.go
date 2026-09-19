package format

import (
	"slices"
	"strings"
)

// AppFormat names the format Illarin writes for one app.
type AppFormat struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Format string `json:"format"`
}

// AppFormats picks the offered format that lands most of the work in each app.
func AppFormats(offered []Offered, r *Registry) []AppFormat {
	picked := make([]AppFormat, 0, len(Apps()))
	for _, app := range Apps() {
		best := -1
		for i := range offered {
			if !slices.Contains(app.Reads, offered[i].Format) {
				continue
			}
			if best < 0 || landsBetter(offered[i], offered[best], app.ID, r) {
				best = i
			}
		}
		if best < 0 {
			continue
		}
		picked = append(picked, AppFormat{
			ID: app.ID, Label: app.Label, Format: offered[best].Format,
		})
	}
	return picked
}

func landsBetter(candidate, holder Offered, app string, r *Registry) bool {
	for _, comparison := range []int{
		holder.LossesFor(app) - candidate.LossesFor(app),
		holder.Notes() - candidate.Notes(),
		writableRoles(r, candidate.Format) - writableRoles(r, holder.Format),
		strings.Compare(holder.Format, candidate.Format),
	} {
		if comparison != 0 {
			return comparison > 0
		}
	}
	return false
}

// PrivatePromptFormats lists the formats the app reads that keep a work's private prompts private.
func (r *Registry) PrivatePromptFormats(app string) []string {
	formats := []string{}
	for _, known := range Apps() {
		if known.ID != app {
			continue
		}
		for _, id := range known.Reads {
			if declaration, ok := r.Declaration(id); ok && declaration.KeepsPrivatePrompts {
				formats = append(formats, id)
			}
		}
	}
	return formats
}

// OfferedIDs lists the format ids of the offered formats.
func OfferedIDs(offered []Offered) []string {
	ids := make([]string, len(offered))
	for i, one := range offered {
		ids[i] = one.Format
	}
	return ids
}
