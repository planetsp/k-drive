package main

import (
	log "github.com/planetsp/k-drive/pkg/logging"

	"github.com/planetsp/k-drive/pkg/sync"
	"github.com/planetsp/k-drive/pkg/ui"
)

func main() {
	log.Info("Starting k-drive")
	ui.RunUI()
	sync.StartSyncClient()
}
