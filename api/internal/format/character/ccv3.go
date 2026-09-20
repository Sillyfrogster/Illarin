package character

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

type CCv3Module struct{}

func (CCv3Module) ID() string { return V3 }

func (CCv3Module) Declaration() format.Declaration { return declaration(V3) }

func (m CCv3Module) Match(file format.Inspection) (format.Match, bool) {
	return format.MatchByDeclaration(file, m.Declaration())
}

func (m CCv3Module) Parse(
	_ context.Context,
	file format.Inspection,
	match format.Match,
) (format.Parsed, error) {
	read, err := readCard(file, match, 3, m.ID())
	if err != nil {
		return format.Parsed{}, err
	}
	return read.parsed(m.ID(), documentImage(file))
}
