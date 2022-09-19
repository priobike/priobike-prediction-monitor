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
type StatusSummary struct {
	// The time of the status update.
	StatusUpdateTime int64 `json:"status_update_time"`
	// The number of things.
	NumThings int `json:"num_things"`
	// The number of predictions.
	NumPredictions int `json:"num_predictions"`
	// The number of predictions with quality <= 0.5.
	NumBadPredictions int `json:"num_bad_predictions"`
	// The time of the most recent prediction.
	MostRecentPredictionTime *int64 `json:"most_recent_prediction_time"`
	// The time of the oldest prediction.
	OldestPredictionTime *int64 `json:"oldest_prediction_time"`
	// The average prediction quality.
	AveragePredictionQuality *float64 `json:"average_prediction_quality"`
}

// Create a summary of the predictions, i.e. whether they are up to date.
// Write the result to a static directory as json.
func WriteSummary() {
	// Fetch the path under which we will save the json files.
	staticPath := os.Getenv("STATIC_PATH")
	if staticPath == "" {
		panic("STATIC_PATH not set")
	}

	numThings := len(sync.Things)
	numPredictions := len(predictions.Current)

	var mostRecentPredictionTime *int64 = nil
	var oldestPredictionTime *int64 = nil
	for _, timestamp := range predictions.Timestamps {
		if mostRecentPredictionTime == nil || timestamp > *mostRecentPredictionTime {
			mostRecentPredictionTime = &timestamp
		}
		if oldestPredictionTime == nil || timestamp < *oldestPredictionTime {
			oldestPredictionTime = &timestamp
		}
	}

	// Calculate the average prediction quality and the number of bad predictions.
	var averagePredictionQuality *float64 = nil
	numBadPredictions := 0
	if numPredictions > 0 {
		var sum float64 = 0
		for _, prediction := range predictions.Current {
			if prediction.PredictionQuality <= 0.5 {
				numBadPredictions++
			}
			if (prediction.PredictionQuality < 0) || (prediction.PredictionQuality > 1) {
				continue
			}
			sum += prediction.PredictionQuality
		}
		average := sum / float64(numPredictions)
		averagePredictionQuality = &average
	}

	// Write the status update to a json file.
	summary := StatusSummary{
		StatusUpdateTime:         time.Now().Unix(),
		NumThings:                numThings,
		NumPredictions:           numPredictions,
		NumBadPredictions:        numBadPredictions,
		MostRecentPredictionTime: mostRecentPredictionTime,
		OldestPredictionTime:     oldestPredictionTime,
		AveragePredictionQuality: averagePredictionQuality,
	}

	// Write the status update to the file.
	statusJson, err := json.Marshal(summary)
	if err != nil {
		log.Error.Println("Error marshalling status summary:", err)
		return
	}
	ioutil.WriteFile(staticPath+"status.json", statusJson, 0644)
}
