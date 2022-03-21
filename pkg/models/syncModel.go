package models

import (
	"time"
)

type FileLocation int
type SyncStatus int

const ( // iota is reset to 0
	Cloud FileLocation = iota // c0 == 0
	Local FileLocation = iota // c1 == 1
)
const (
	Synced      SyncStatus = iota // c0 == 0
	Uploading   SyncStatus = iota // c1 == 1
	Downloading SyncStatus = iota // c1 == 1
)

// Todo use date and time to decide who
type SyncInfo struct {
	Filename     string
	DateModified time.Time
	Location     FileLocation
	SyncStatus   SyncStatus
}

type SyncDiff struct {
	FilesNotAvailableInCloud []SyncInfo
	FilesNotAvailableLocally []SyncInfo
}

func CreateSyncInfo(filename string, dateModified time.Time, location FileLocation, syncStatus SyncStatus) *SyncInfo {
	return &SyncInfo{
		Filename:     filename,
		DateModified: dateModified,
		SyncStatus:   syncStatus,
		Location:     location,
	}
}

func (sS SyncStatus) String() string {
	if sS == Uploading {
		return "Uploading"
	} else if sS == Downloading {
		return "Downloading"
	} else if sS == Synced {
		return "Synced"
	}
	return "Unknown"
}
func (loc FileLocation) String() string {
	if loc == Cloud {
		return "Cloud"
	} else if loc == Local {
		return "Local"
	}
	return "Unknown"
}
