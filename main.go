package main

import (
	"monitor/history"
	"monitor/log"
	"monitor/predictions"
	"monitor/status"
	"monitor/sync"
)

func main() {
	log.Init()

	// Start the sync service.
	go sync.Run()

	// Start the prediction listener.
	go predictions.Listen()

	// Monitor the status of the predictions.
	go status.Monitor()

	// Start the sync of the history.
	go history.Sync()

	// Wait forever.
	select {}
}
