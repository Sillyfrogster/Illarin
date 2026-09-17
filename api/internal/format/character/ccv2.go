package character

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

type CCv2Module struct{}

func (CCv2Module) ID() string { return V2 }

func (CCv2Module) Declaration() format.Declaration { return declaration(V2) }

func (m CCv2Module) Claim(file format.Inspection) (format.Claim, bool) {
	return format.ClaimByDeclaration(file, m.Declaration())
}

func (m CCv2Module) Parse(
	_ context.Context,
	file format.Inspection,
	claim format.Claim,
) (format.Parsed, error) {
	read, err := readCard(file, claim, 2, m.ID())
	if err != nil {
		return format.Parsed{}, err
	}
	return read.parsed(m.ID(), documentImage(file))
}
