package http

type DownloadExportParams struct {
	Images  *string `json:"images,omitempty"`
	Version *int    `json:"version,omitempty"`
}

type GetMediaVariantParams struct {
	Expires   *string `json:"expires,omitempty"`
	Signature *string `json:"signature,omitempty"`
}
