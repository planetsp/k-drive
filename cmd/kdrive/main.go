package main

import (
	"log"
	"time"

	"github.com/planetsp/k-drive/pkg/ui"
)

var tableData = [][]string{
	[]string{"filename", "date modified", "location", "status"}}

type FileLocation int
type SyncStatus int

const ( // iota is reset to 0
	Cloud FileLocation = iota // c0 == 0
	Local              = iota // c1 == 1
)
const (
	Synced      SyncStatus = iota // c0 == 0
	Uploading              = iota // c1 == 1
	Downloading            = iota // c1 == 1
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

func main() {
	log.Println("Starting k-drive")
	ui.RunUI()
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
	}
	return "fasdsdfafdsa"
}
func AddSyncInfoToFyneTable(syncInfo *SyncInfo) {
	slice := []string{syncInfo.Filename, syncInfo.DateModified.String(), syncInfo.SyncStatus.String()}
	tableData = append(tableData, slice)
}
