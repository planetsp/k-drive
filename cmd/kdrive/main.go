package main

import (
	log "github.com/planetsp/k-drive/pkg/logging"
	s "github.com/planetsp/k-drive/pkg/models"

	"github.com/planetsp/k-drive/pkg/sync"
	"github.com/planetsp/k-drive/pkg/ui"
)

func main() {
	log.Info("Starting k-drive")
	syncInfoChannel := make(chan *s.SyncInfo)

	go sync.StartSyncClient(syncInfoChannel)
	ui.RunUI()
}
