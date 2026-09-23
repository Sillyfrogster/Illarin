package format

import (
	"context"
	"io"
	"slices"
)

type Module interface {
	ID() string
	Declaration() Declaration
}

type Reader interface {
	Module
	Match(Inspection) (Match, bool)
	Parse(ctx context.Context, file Inspection, match Match) (Parsed, error)
}

type DatabaseReader interface {
	Module
	ReadDatabaseRow(ctx context.Context, row any) (Parsed, error)
}

type SpecOwner interface {
	OwnedSpecs() []string
}

func ownsSpec(module Module, spec string) bool {
	if spec == module.ID() {
		return true
	}
	owner, ok := module.(SpecOwner)
	return ok && slices.Contains(owner.OwnedSpecs(), spec)
}

type matchStrength uint8

const (
	compatibility matchStrength = iota + 1
	authoritative
)

type Match struct {
	payloadID uint32
	strength  matchStrength
	formatID  string
	byteSize  int64
}

const wholeFilePayloadID = ^uint32(0)

func WholeFileCompatibilityMatch(file Inspection) Match {
	return Match{
		payloadID: wholeFilePayloadID, strength: compatibility, byteSize: file.ByteSize(),
	}
}

func AuthoritativeMatch(payload Payload, marker string) (Match, bool) {
	formatID, ok := payload.String(marker)
	if !ok || formatID == "" {
		return Match{}, false
	}
	return Match{payloadID: payload.ID, strength: authoritative, formatID: formatID}, true
}

func CompatibilityMatch(payload Payload) Match {
	return Match{payloadID: payload.ID, strength: compatibility}
}

func (c Match) Payload(file Inspection) (Payload, bool) {
	if c.payloadID == wholeFilePayloadID {
		return Payload{ID: wholeFilePayloadID, ByteSize: c.byteSize}, true
	}
	for _, payload := range file.Payloads {
		if payload.ID == c.payloadID {
			return payload, true
		}
	}
	return Payload{}, false
}

type LabelledText struct {
	Label string
	Text  string
}

type TextExtractor interface {
	ExtractText(ctx context.Context, src io.Reader) ([]LabelledText, error)
}
