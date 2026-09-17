package notify

type AssetWatch struct {
	InstalledOn []string        `json:"installedOn"`
	State       AssetWatchState `json:"state"`
}

type AssetWatchState string

const (
	AssetWatchStateInstalled AssetWatchState = "installed"
	AssetWatchStateNone      AssetWatchState = "none"
	AssetWatchStateStopped   AssetWatchState = "stopped"
	AssetWatchStateWatching  AssetWatchState = "watching"
)
