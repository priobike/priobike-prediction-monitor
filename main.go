package main

import (
	"encoding/json"
	predictions "monitor/predictions"
	sync "monitor/sync"
	"os"

	"fmt"
	"io/ioutil"
	"time"

	geojson "github.com/paulmach/go.geojson"
)

// A status update that is written to json.
type StatusUpdate struct {
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

// Check the status of the predictions, i.e. whether they are up to date.
func checkStatus() {
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
	statusUpdate := StatusUpdate{
		StatusUpdateTime:         time.Now().Unix(),
		NumThings:                numThings,
		NumPredictions:           numPredictions,
		NumBadPredictions:        numBadPredictions,
		MostRecentPredictionTime: mostRecentPredictionTime,
		OldestPredictionTime:     oldestPredictionTime,
		AveragePredictionQuality: averagePredictionQuality,
	}

	// Get the status file path from the environment.
	statusFilePath := os.Getenv("STATUS_FILE_PATH")
	if statusFilePath == "" {
		fmt.Println("STATUS_FILE_PATH not set. Skipping status file update.")
		return
	}

	// Write the status update to the file.
	statusJson, err := json.Marshal(statusUpdate)
	if err != nil {
		fmt.Println(err)
		return
	}
	ioutil.WriteFile(statusFilePath, statusJson, 0644)
}

// Write a geojson file for all things.
func writeGeoJson() {
	// Get the geojson file path from the environment.
	geoJsonFilePath := os.Getenv("GEOJSON_FILE_PATH")
	if geoJsonFilePath == "" {
		fmt.Println("GEOJSON_FILE_PATH not set. Skipping geojson file update.")
		return
	}

	// Write the geojson to the file.
	featureCollection := geojson.NewFeatureCollection()
	for _, thing := range sync.Things {
		lat, lng, err := thing.LatLng()
		if err != nil {
			continue
		}
		// Check if there is a prediction for this thing.
		prediction, predictionOk := predictions.Current[thing.Topic()]
		// Check the time diff between the prediction and the current time.
		predictionTime, predictionTimeOk := predictions.Timestamps[thing.Topic()]
		feature := geojson.NewPointFeature([]float64{*lng, *lat})
		if predictionOk && predictionTimeOk {
			feature.Properties = map[string]interface{}{
				"prediction_available": true,
				"prediction_quality":   prediction.PredictionQuality,
				"prediction_time_diff": time.Now().Unix() - predictionTime,
				"prediction_sg_id":     prediction.SignalGroupId,
			}
		} else {
			feature.Properties = map[string]interface{}{
				"prediction_available": false,
				"prediction_quality":   0,
				"prediction_time_diff": 0,
				"prediction_sg_id":     "",
			}
		}
		featureCollection.AddFeature(feature)
	}
	geoJson, err := featureCollection.MarshalJSON()
	if err != nil {
		fmt.Println("Error marshalling geojson:", err)
		return
	}

	ioutil.WriteFile(geoJsonFilePath, geoJson, 0644)
}

// Continuously log out interesting things.
func monitor() {
	// Wait a bit initially to let the sync service do its job.
	time.Sleep(20 * time.Second)
	for {
		fmt.Println("Calculating metrics...")

		// Check the status of the predictions.
		checkStatus()

		// Write the geojson file.
		writeGeoJson()

		fmt.Println("Finished calculating metrics.")
		// Sleep for 1 minute.
		time.Sleep(1 * time.Minute)
	}
}

func main() {
	fmt.Println("Starting monitoring service...")

	// Start the sync service.
	go sync.Run()

	// Start the prediction listener.
	go predictions.Listen()

	// Start the monitoring service.
	go monitor()

	// Wait forever.
	select {}
}
