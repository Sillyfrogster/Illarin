package http

type ImportPostMarkdownRequest struct {
	Markdown string `json:"markdown"`
	Version  int    `json:"version"`
}

type PostImport struct {
	Post     Post             `json:"post"`
	Warnings []PostImportNote `json:"warnings"`
}

type PostImportNote struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type PostImportRefusal struct {
	Code     PublicationErrorCode `json:"code"`
	Error    string               `json:"error"`
	Field    *string              `json:"field,omitempty"`
	Refusals *[]PostImportNote    `json:"refusals,omitempty"`
}
