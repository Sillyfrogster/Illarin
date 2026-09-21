package download

import (
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
)

// FormatTable reads every written format's declaration into one comparison, field by field
func FormatTable(r *format.Registry) FormatComparison {
	apps := format.Apps()
	ids := make([]string, 0, len(apps))
	for _, app := range apps {
		ids = append(ids, app.ID)
	}
	written := []format.Declaration{}
	for _, declaration := range r.Declarations() {
		if declaration.Direction.Write {
			written = append(written, declaration)
		}
	}
	columns := make([]FormatColumn, 0, len(written))
	for _, declaration := range written {
		columns = append(columns, FormatColumn{
			Id: declaration.ID, Label: declaration.Label, Type: declaration.Type,
			ReadBy:      format.AppsReading([]string{declaration.ID}),
			KeepsUpload: declaration.KeepsUpload,
			Fields:      fieldSupport(declaration, typeFields(written, declaration.Type)),
		})
	}
	return FormatComparison{Apps: page.AppNames(ids), Formats: columns}
}

// typeFields lists the fields any format of the type declares, in field order
func typeFields(written []format.Declaration, workType string) []block.Role {
	fields := []block.Role{}
	for _, role := range block.Roles() {
		for _, declaration := range written {
			if declaration.Type == workType && !declaration.KeepsUpload && declaration.Roles[role].Write.Grade != "" {
				fields = append(fields, role)
				break
			}
		}
	}
	return fields
}

func fieldSupport(declaration format.Declaration, fields []block.Role) []FieldSupport {
	if declaration.KeepsUpload {
		return []FieldSupport{}
	}
	support := make([]FieldSupport, 0, len(fields))
	for _, role := range fields {
		write := declaration.Roles[role].Write
		grade := write.Grade
		if grade == "" {
			grade = format.SupportNone
		}
		support = append(support, FieldSupport{
			Field: role, Label: role.Label(), Grade: string(grade), Note: supportNote(write),
		})
	}
	return support
}

func supportNote(write format.RoleSupport) string {
	notes := []string{}
	if write.Grade == format.SupportPartial && write.Condition != nil {
		notes = append(notes, "Loses "+write.Condition.Description+".")
	}
	if write.Grade != format.SupportNone && write.Destination != "" {
		notes = append(notes, write.Destination)
		if len(write.ShownBy) > 0 {
			labels := make([]string, 0, len(write.ShownBy))
			for _, id := range write.ShownBy {
				labels = append(labels, format.AppLabel(id))
			}
			verb := "shows"
			if len(labels) > 1 {
				verb = "show"
			}
			notes = append(notes, "Only "+strings.Join(labels, " and ")+" "+verb+" these.")
		}
	}
	return strings.Join(slices.Compact(notes), " ")
}
