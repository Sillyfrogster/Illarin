package upload

type StartAssetRequest struct {
	App  *StartAssetRequestApp `json:"app,omitempty"`
	Kind string                `json:"kind"`
}

type StartAssetRequestApp string

const (
	StartAssetRequestAppLumiverse   StartAssetRequestApp = "lumiverse"
	StartAssetRequestAppSillytavern StartAssetRequestApp = "sillytavern"
)
