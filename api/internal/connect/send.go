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
	ErrNoAppOfYours      = errors.New("no live connected app of yours has that id")
	ErrWorkNotFound      = errors.New("no such work")
	ErrWorkNotSendable   = errors.New("only a published work can be sent to a connected app")
	ErrNoFormat          = errors.New("that connected app accepts no format this work can be written in")
	ErrCannotInstall     = errors.New("that connected app does not install what this work is")
	ErrQueueFull         = errors.New("that connected app already has as many waiting sends as it may hold")
	ErrSendNotFound      = errors.New("no waiting send of yours has that id")
	ErrTooManyCollectors = errors.New("too many connected apps are waiting for work at once")
	ErrLibraryTooLarge   = errors.New("the library report is larger than one request may carry")
	ErrLibraryReport     = errors.New("the library report is not valid")
	ErrLibraryVersion    = errors.New("the library report names an app version that is not short printable text")
	ErrAcknowledgement   = errors.New("the acknowledgement list is not valid")
)

type Send struct {
	ID             uuid.UUID
	ConnectedAppID uuid.UUID
	WorkID         uuid.UUID
	State          State
	Reason         Reason
	QueuedAt       time.Time
	SettledAt      *time.Time
	ExpiresAt      time.Time
	UpdatesInstall bool
}

type Work struct {
	ID             uuid.UUID
	WorkID         uuid.UUID
	VersionNumber  int
	Type           string
	Name           string
	Format         string
	Label          string
	QueuedAt       time.Time
	LeaseExpiresAt time.Time
	Files          []File
}

type File struct {
	Type    string
	URL     string
	MediaID *uuid.UUID
	Role    string
	IsCover bool
}

const (
	FileExport  = "export"
	FilePicture = "picture"
)

type AppState struct {
	ConnectedAppID   uuid.UUID
	AppName          string
	Name             string
	LastSeenAt       *time.Time
	CanReceive       bool
	ReportsLibrary   bool
	Send             *Send
	InstalledVersion *int
	UpdateAvailable  bool
}

type WorkApps struct {
	VersionNumber int
	Items         []AppState
}

type LibraryCounts struct {
	Installed        int
	UpdatesAvailable int
}

type ReportedLibrary struct {
	Snapshot   bool
	AppVersion string
	Entries    []ReportedEntry
	Removed    []uuid.UUID
}

type ReportedEntry struct {
	WorkID        uuid.UUID
	VersionNumber *int
}

type LibraryResult struct {
	Accepted  int
	Removed   int
	Ignored   int
	Takedowns []TakenDownWork
}

type TakenDownWork struct {
	WorkID      uuid.UUID
	Name        string
	TakenDownAt time.Time
}

type Collected struct {
	Work      []Work
	Takedowns []TakenDownWork
}

func chooseFormat(accepted []string, offered []SendFormat, hasOriginal bool) (string, string, bool) {
	byID := make(map[string]SendFormat, len(offered))
	for _, one := range offered {
		byID[one.Format] = one
	}
	for _, wanted := range accepted {
		if one, offers := byID[wanted]; offers {
			return one.Format, one.Label, true
		}
		if wanted == format.Raw && hasOriginal {
			return format.Raw, rawLabel, true
		}
	}
	if hasOriginal {
		return format.Raw, rawLabel, true
	}
	return "", "", false
}

const rawLabel = "The creator's own file"

// installs says whether the connected app declared one of the capabilities the work needs, or the work needs none.
func installs(declared []string, sendable Sendable) bool {
	if len(sendable.InstallCapabilities) == 0 {
		return true
	}
	for _, needed := range sendable.InstallCapabilities {
		if slices.Contains(declared, needed) {
			return true
		}
	}
	return false
}
