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
	Claim(Inspection) (Claim, bool)
	Parse(ctx context.Context, file Inspection, claim Claim) (Parsed, error)
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

type claimStrength uint8

const (
	compatibility claimStrength = iota + 1
	authoritative
)

type Claim struct {
	payloadID uint32
	strength  claimStrength
	formatID  string
	byteSize  int64
}

const wholeFilePayloadID = ^uint32(0)

func WholeFileCompatibilityClaim(file Inspection) Claim {
	return Claim{
		payloadID: wholeFilePayloadID, strength: compatibility, byteSize: file.ByteSize(),
	}
}

func AuthoritativeClaim(payload Payload, discriminator string) (Claim, bool) {
	formatID, ok := payload.String(discriminator)
	if !ok || formatID == "" {
		return Claim{}, false
	}
	return Claim{payloadID: payload.ID, strength: authoritative, formatID: formatID}, true
}

func CompatibilityClaim(payload Payload) Claim {
	return Claim{payloadID: payload.ID, strength: compatibility}
}

func (c Claim) Payload(file Inspection) (Payload, bool) {
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
