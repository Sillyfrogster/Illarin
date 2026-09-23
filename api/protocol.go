package api

import _ "embed"

// Protocol points older /protocol readers to the maintained documentation.
//
//go:embed protocol.md
var Protocol []byte
