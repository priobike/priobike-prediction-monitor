package main

import (
	"monitor/log"
	"monitor/predictions"
	"monitor/status"
	"monitor/sync"
	"monitor/worker"
	"os"
)

func main() {
	log.Init()

	workerModeStr := os.Getenv("WORKER_MODE")
	if workerModeStr == "" {
		panic("WORKER_MODE not set")
	}
	workerMode := workerModeStr == "true"

	if workerMode {
		// -- Start as worker. --
		go worker.Run()
	} else {
		// -- Start as manager. --

		// Start the sync service.
		go sync.Run()

		// Start the prediction listener.
		go predictions.Listen()

		// Monitor the status of the predictions.
		go status.Monitor()
	}

	// Wait forever.
	select {}
}
