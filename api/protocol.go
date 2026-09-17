package api

import _ "embed"

// Protocol is the guide for apps that connect to Illarin, served at /protocol
//
//go:embed protocol.md
var Protocol []byte
