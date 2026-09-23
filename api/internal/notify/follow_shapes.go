package notify

type WorkFollow struct {
	InstalledOn []string        `json:"installedOn"`
	State       WorkFollowState `json:"state"`
}

type WorkFollowState string

const (
	WorkFollowStateInstalled WorkFollowState = "installed"
	WorkFollowStateNone      WorkFollowState = "none"
	WorkFollowStateStopped   WorkFollowState = "stopped"
	WorkFollowStateFollowing WorkFollowState = "following"
)
