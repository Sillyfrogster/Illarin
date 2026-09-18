package connect

import (
	"errors"
	"slices"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

type State string

const (
	StateQueued    State = "queued"
	StateReleased  State = "released"
	StateDelivered State = "delivered"
	StateFailed    State = "failed"
)

type Reason string

const (
	ReasonWithdrawn   Reason = "withdrawn"
	ReasonUnsupported Reason = "unsupported"
	ReasonAbandoned   Reason = "abandoned"
)

var (
	ErrNoInstanceOfYours = errors.New("no live instance of yours has that id")
	ErrMissingScope      = errors.New("that instance was not granted the scope this needs")
	ErrWorkNotFound      = errors.New("no such work")
	ErrWorkNotSendable   = errors.New("only a published work can be sent to an instance")
	ErrNoTarget          = errors.New("that instance accepts no format this work can be written in")
	ErrCannotInstall     = errors.New("that instance does not install what this work is")
	ErrQueueFull         = errors.New("that instance already has as many waiting deliveries as it may hold")
	ErrDeliveryNotFound  = errors.New("no waiting delivery of yours has that id")
	ErrTooManyCollectors = errors.New("too many instances are waiting for work at once")
	ErrLibraryTooLarge   = errors.New("the library report is larger than one request may carry")
	ErrLibraryReport     = errors.New("the library report is not valid")
	ErrLibraryVersion    = errors.New("the library report names an application version that is not short printable text")
	ErrAcknowledgement   = errors.New("the acknowledgement list is not valid")
)

type Delivery struct {
	ID             uuid.UUID
	InstanceID     uuid.UUID
	WorkID         uuid.UUID
	State          State
	Reason         Reason
	QueuedAt       time.Time
	SettledAt      *time.Time
	ExpiresAt      time.Time
	UpdatesInstall bool
}

type Work struct {
	ID                uuid.UUID
	WorkID            uuid.UUID
	ContentGeneration int
	Type              string
	Name              string
	Format            string
	Label             string
	QueuedAt          time.Time
	LeaseExpiresAt    time.Time
	Artifacts         []Artifact
}

type Artifact struct {
	Type    string
	URL     string
	MediaID *uuid.UUID
	Role    string
	IsCover bool
}

const (
	ArtifactExport  = "export"
	ArtifactPicture = "picture"
)

type InstanceState struct {
	InstanceID          uuid.UUID
	ApplicationName     string
	InstanceName        string
	LastSeenAt          *time.Time
	CanReceive          bool
	ReportsLibrary      bool
	Delivery            *Delivery
	InstalledGeneration *int
	UpdateAvailable     bool
}

type WorkInstances struct {
	ContentGeneration int
	Items             []InstanceState
}

type LibraryCounts struct {
	Installed        int
	UpdatesAvailable int
}

type ReportedLibrary struct {
	Snapshot           bool
	ApplicationVersion string
	Entries            []ReportedEntry
	Removed            []uuid.UUID
}

type ReportedEntry struct {
	WorkID            uuid.UUID
	ContentGeneration *int
}

type LibraryResult struct {
	Accepted int
	Removed  int
	Ignored  int
	Withheld []WithheldWork
}

type WithheldWork struct {
	WorkID     uuid.UUID
	Name       string
	WithheldAt time.Time
}

type Collected struct {
	Work     []Work
	Withheld []WithheldWork
}

func chooseTarget(accepted []string, offered []DeliveryTarget, hasOriginal bool) (string, string, bool) {
	byID := make(map[string]DeliveryTarget, len(offered))
	for _, target := range offered {
		byID[target.Format] = target
	}
	for _, wanted := range accepted {
		if target, offers := byID[wanted]; offers {
			return target.Format, target.Label, true
		}
		if wanted == format.RawTarget && hasOriginal {
			return format.RawTarget, rawLabel, true
		}
	}
	if hasOriginal {
		return format.RawTarget, rawLabel, true
	}
	return "", "", false
}

const rawLabel = "The creator's own file"

// installs says whether the instance declared one of the capabilities the work needs, or the work needs none.
func installs(capabilities []string, sendable Deliverable) bool {
	if len(sendable.InstallCapabilities) == 0 {
		return true
	}
	for _, needed := range sendable.InstallCapabilities {
		if slices.Contains(capabilities, needed) {
			return true
		}
	}
	return false
}
