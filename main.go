package main

import (
	"encoding/json"
	log "monitor/log"
	predictions "monitor/predictions"
	sync "monitor/sync"
	"os"

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
		log.Warning.Println("STATUS_FILE_PATH not set. Skipping status file update.")
		return
	}

	// Write the status update to the file.
	statusJson, err := json.Marshal(statusUpdate)
	if err != nil {
		log.Error.Println("Error marshalling status update:", err)
		return
	}
	ioutil.WriteFile(statusFilePath, statusJson, 0644)
}

// Write a geojson file for all things.
func writeGeoJson() {
	// Get the geojson file paths from the environment.
	geoJsonLocationsFilePath := os.Getenv("GEOJSON_LOCATIONS_FILE_PATH")
	if geoJsonLocationsFilePath == "" {
		log.Warning.Println("GEOJSON_LOCATIONS_FILE_PATH not set. Skipping geojson file update.")
		return
	}
	geoJsonLanesFilePath := os.Getenv("GEOJSON_LANES_FILE_PATH")
	if geoJsonLanesFilePath == "" {
		log.Warning.Println("GEOJSON_LANES_FILE_PATH not set. Skipping geojson file update.")
		return
	}

	// Write the geojson to the file.
	locationFeatureCollection := geojson.NewFeatureCollection() // Locations of traffic lights.
	laneFeatureCollection := geojson.NewFeatureCollection()     // Lanes of traffic lights.
	for _, thing := range sync.Things {
		lane, err := thing.Lane()
		if err != nil {
			log.Warning.Printf("Error getting lane for thing %s: %v\n", thing.Name, err)
			continue
		}
		coordinate := lane[0]
		lat, lng := coordinate[0], coordinate[1]

		// Check if there is a prediction for this thing.
		prediction, predictionOk := predictions.Current[thing.Topic()]
		// Check the time diff between the prediction and the current time.
		predictionTime, predictionTimeOk := predictions.Timestamps[thing.Topic()]
		// Build the properties.
		properties := make(map[string]interface{})
		if predictionOk && predictionTimeOk {
			properties["prediction_available"] = true
			properties["prediction_quality"] = prediction.PredictionQuality
			properties["prediction_time_diff"] = time.Now().Unix() - predictionTime
			properties["prediction_sg_id"] = prediction.SignalGroupId
		} else {
			properties["prediction_available"] = false
			properties["prediction_quality"] = -1
			properties["prediction_time_diff"] = 0
			properties["prediction_sg_id"] = ""
		}
		// Add thing-related properties.
		properties["thing_name"] = thing.Name
		properties["thing_properties_lanetype"] = thing.Properties.LaneType

		// Make a point feature.
		location := geojson.NewPointFeature([]float64{lng, lat})
		location.Properties = properties
		locationFeatureCollection.AddFeature(location)

		// Make a line feature.
		laneFeature := geojson.NewLineStringFeature(lane)
		laneFeature.Properties = properties
		laneFeatureCollection.AddFeature(laneFeature)
	}

	locationsGeoJson, err := locationFeatureCollection.MarshalJSON()
	if err != nil {
		log.Error.Println("Error marshalling geojson:", err)
		return
	}
	ioutil.WriteFile(geoJsonLocationsFilePath, locationsGeoJson, 0644)

	lanesGeoJson, err := laneFeatureCollection.MarshalJSON()
	if err != nil {
		log.Error.Println("Error marshalling geojson:", err)
		return
	}
	ioutil.WriteFile(geoJsonLanesFilePath, lanesGeoJson, 0644)
}

// Continuously log out interesting things.
func monitor() {
	log.Info.Println("Starting monitor...")
	// Wait a bit initially to let the sync service do its job.
	time.Sleep(20 * time.Second)
	for {
		log.Info.Println("Running monitor...")

		// Check the status of the predictions.
		checkStatus()

		// Write the geojson file.
		writeGeoJson()

		log.Info.Println("Done running monitor.")
		// Sleep for 1 minute.
		time.Sleep(1 * time.Minute)
	}
}

func main() {
	log.Init()

	// Start the sync service.
	go sync.Run()

	// Start the prediction listener.
	go predictions.Listen()

	// Start the monitoring service.
	go monitor()

	// Wait forever.
	select {}
}
