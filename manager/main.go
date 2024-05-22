package main

import (
	"monitor/log"
	"monitor/predictions"
	"monitor/status"
	"monitor/sync"
)

func main() {
	log.Init()

	// Start the prediction listener.
	// We run this before doing anything else to ensure the prediction broker is online.
	// If the broker is offline, it doesn't make sense to start the sync service.
	predictions.Listen()

	// Start the sync service.
	go sync.Run()

	// Monitor the status of the predictions.
	go status.Monitor()

	// Wait forever.
	select {}
}
