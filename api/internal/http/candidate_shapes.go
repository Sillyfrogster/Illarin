package http

type CandidateConflict struct {
	Code           CandidateConflictCode `json:"code"`
	CurrentVersion *int64                `json:"currentVersion,omitempty"`
	Error          string                `json:"error"`
}

type CandidateConflictCode string

const (
	CandidateConflictCodeAssetFrozen         CandidateConflictCode = "asset_frozen"
	CandidateConflictCodeWorkingCopyConflict CandidateConflictCode = "working_copy_conflict"
)
