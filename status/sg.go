package status

import (
	"encoding/json"
	"io/ioutil"
	"monitor/log"
	"monitor/predictions"
	"monitor/sync"
	"os"
	"time"
)

// A status summary of all predictions that is written to json.
type SGStatus struct {
	// The time of the status update.
	StatusUpdateTime int64 `json:"status_update_time"`
	// The name of the thing.
	ThingName string `json:"thing_name"`
	// The current prediction quality, if there is a prediction.
	PredictionQuality *float64 `json:"prediction_quality"`
	// The unix time of the last prediction, if there is a prediction.
	PredictionTime *int64 `json:"prediction_time"`
}

// Write a status file for each signal group.
func WriteStatusForEachSG() {
	// Fetch the path under which we will save the json files.
	staticPath := os.Getenv("STATIC_PATH")
	if staticPath == "" {
		panic("STATIC_PATH not set")
	}

	// Lock resources.
	sync.ThingsMutex.Lock()
	defer sync.ThingsMutex.Unlock()
	predictions.CurrentMutex.Lock()
	defer predictions.CurrentMutex.Unlock()
	predictions.TimestampsMutex.Lock()
	defer predictions.TimestampsMutex.Unlock()

	for _, thing := range sync.Things {
		// Create the status summary.
		status := SGStatus{
			StatusUpdateTime: time.Now().Unix(),
			ThingName:        thing.Name,
		}

		// Get the prediction for the signal group.
		prediction, ok := predictions.Current[thing.Topic()]
		if ok {
			status.PredictionQuality = &prediction.PredictionQuality
		}

		// Get the prediction time.
		timestamp, ok := predictions.Timestamps[thing.Topic()]
		if ok {
			status.PredictionTime = &timestamp
		}

		// Write the status update to a json file.
		statusJson, err := json.Marshal(status)
		if err != nil {
			log.Error.Println("Error marshalling status:", err)
			continue
		}
		path := staticPath + thing.Topic()
		if err := ioutil.WriteFile(path+"/status.json", statusJson, 0644); err != nil {
			// If the path contains a directory that does not exist, create it.
			// But don't create a folder for the file itself.
			if err := os.MkdirAll(path, 0755); err != nil {
				log.Error.Println("Error creating directory for status file:", err)
				continue
			}
		}
	}
}
