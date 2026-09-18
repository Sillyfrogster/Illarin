package apitest

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

type ClaimsFirstPayload struct{}

func (ClaimsFirstPayload) Claim(file format.Inspection) (format.Claim, bool) {
	if len(file.Payloads) == 0 {
		return format.Claim{}, false
	}
	return format.CompatibilityClaim(file.Payloads[0]), true
}

type RecognizedModule struct {
	ClaimsFirstPayload
	Parsed format.Parsed
}

func (RecognizedModule) ID() string { return "recognized" }
func (RecognizedModule) Declaration() format.Declaration {
	return ReaderDeclaration("recognized", "character")
}
func (module RecognizedModule) Parse(context.Context, format.Inspection, format.Claim) (format.Parsed, error) {
	return module.Parsed, nil
}

type ReplacingModule struct {
	ClaimsFirstPayload
	Parsed *format.Parsed
}

func (ReplacingModule) ID() string { return "replacing" }
func (ReplacingModule) Declaration() format.Declaration {
	declaration := ReaderDeclaration("replacing", "character")
	declaration.Label = "Replacing format"
	declaration.Direction.Write = true
	declaration.Header = []format.HeaderField{format.HeaderName, format.HeaderWorkVersion}
	declaration.TestedOrigins = append(declaration.TestedOrigins, format.OriginIllarin)
	declaration.Roles = map[block.Role]format.DirectionalRoleSupport{
		block.RoleDescription: {
			Read:  format.RoleSupport{Grade: format.SupportFull},
			Write: format.RoleSupport{Grade: format.SupportFull},
		},
	}
	return declaration
}
func (module ReplacingModule) Parse(context.Context, format.Inspection, format.Claim) (format.Parsed, error) {
	return *module.Parsed, nil
}
func (ReplacingModule) Write(context.Context, format.ExportWork) (format.Artifact, error) {
	return format.Artifact{MediaType: "text/plain", Extension: ".txt"}, nil
}
