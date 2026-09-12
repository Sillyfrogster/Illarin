package format

import (
	"slices"
	"strings"
)

// AppTarget names the format Illarin writes for one application.
type AppTarget struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Format string `json:"format"`
}

// AppTargets picks the offered format that lands most of the asset in each application.
func AppTargets(targets []Target, r *Registry) []AppTarget {
	picked := make([]AppTarget, 0, len(Apps()))
	for _, app := range Apps() {
		best := -1
		for i := range targets {
			if !slices.Contains(app.Reads, targets[i].Format) {
				continue
			}
			if best < 0 || landsBetter(targets[i], targets[best], app.ID, r) {
				best = i
			}
		}
		if best < 0 {
			continue
		}
		picked = append(picked, AppTarget{
			ID: app.ID, Label: app.Label, Format: targets[best].Format,
		})
	}
	return picked
}

func landsBetter(candidate, holder Target, app string, r *Registry) bool {
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
