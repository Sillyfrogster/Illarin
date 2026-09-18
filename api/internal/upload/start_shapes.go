package upload

type StartWorkRequest struct {
	App  *StartWorkRequestApp `json:"app,omitempty"`
	Type string               `json:"type"`
}

type StartWorkRequestApp string

const (
	StartWorkRequestAppLumiverse   StartWorkRequestApp = "lumiverse"
	StartWorkRequestAppSillytavern StartWorkRequestApp = "sillytavern"
)
