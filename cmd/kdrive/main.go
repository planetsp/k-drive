package main

import (
	c "github.com/planetsp/k-drive/pkg/config"
	log "github.com/planetsp/k-drive/pkg/logging"
	s "github.com/planetsp/k-drive/pkg/models"

	"github.com/planetsp/k-drive/pkg/sync"
	"github.com/planetsp/k-drive/pkg/ui"
)

func main() {
	log.Info("Starting k-drive")

	syncInfoChannel := make(chan *s.SyncInfo)

	// Wait for configuration to be ready and start sync client in background
	go func() {
		<-ui.GetConfigReadyChannel()
		log.Info("Configuration ready, starting sync client")

		// Check if config is valid
		if !c.IsConfigLoaded() {
			log.Error("Configuration not loaded properly")
			return
		}

		config := c.GetConfig()
		if config.WorkingDirectory == "" || config.BucketName == "" {
			log.Error("Invalid configuration: missing required fields")
			return
		}

		// Start sync client only after config is ready
		go sync.StartSyncClient(syncInfoChannel)

		// Handle sync info updates
		for syncInfo := range syncInfoChannel {
			log.Info("Sync update: %s - %s", syncInfo.Filename, syncInfo.SyncStatus.String())
			ui.AddSyncInfoToFyneTable(syncInfo)
		}
	}()

	// Run UI in main goroutine (required by Fyne)
	ui.RunUI()
}
