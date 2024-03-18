package main

import (
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

	// Wait forever.
	select {}
}
