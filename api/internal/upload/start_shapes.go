package upload

type StartWorkRequest struct {
	App  *StartWorkRequestApp `json:"app,omitempty"`
	Type string               `json:"kind"`
}

type StartWorkRequestApp string

const (
	StartWorkRequestAppLumiverse   StartWorkRequestApp = "lumiverse"
	StartWorkRequestAppSillytavern StartWorkRequestApp = "sillytavern"
)
