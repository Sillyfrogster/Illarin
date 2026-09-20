package character

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

type CCv2Module struct{}

func (CCv2Module) ID() string { return V2 }

func (CCv2Module) Declaration() format.Declaration { return declaration(V2) }

func (m CCv2Module) Match(file format.Inspection) (format.Match, bool) {
	return format.MatchByDeclaration(file, m.Declaration())
}

func (m CCv2Module) Parse(
	_ context.Context,
	file format.Inspection,
	match format.Match,
) (format.Parsed, error) {
	read, err := readCard(file, match, 2, m.ID())
	if err != nil {
		return format.Parsed{}, err
	}
	return read.parsed(m.ID(), documentImage(file))
}
