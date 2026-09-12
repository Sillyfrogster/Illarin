package openapi

import _ "embed"

//go:embed openapi.gen.yaml
var Contract []byte

//go:embed protocol.md
var Guide []byte
