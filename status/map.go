package status

import (
	"fmt"
	"io/ioutil"
	"monitor/log"
	"monitor/predictions"
	"monitor/sync"
	"os"
	"time"

	geojson "github.com/paulmach/go.geojson"
)

// Write geojson data that can be used to visualize the predictions.
// The geojson file is written to the static directory.
func WriteGeoJSONMap() {
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

	// Write the geojson to the file.
	locationFeatureCollection := geojson.NewFeatureCollection() // Locations of traffic lights.
	laneFeatureCollection := geojson.NewFeatureCollection()     // Lanes of traffic lights.

	// Create another list that will contain Prometheus metrics for each thing.
	// In this way we can visualize the metrics in Grafana, on a map.
	metrics := make([]string, 0)

	for _, thing := range sync.Things {
		lane, err := thing.Lane()
		if err != nil {
			log.Warning.Printf("Error getting lane for thing %s: %v\n", thing.Name, err)
			continue
		}
		coordinate := lane[0]
		lat, lng := coordinate[1], coordinate[0]

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

		// Make a prometheus metric containing all the properties.
		metric := "prediction_monitor_prediction{"
		metric += fmt.Sprintf("lat=\"%v\",lng=\"%v\",", lat, lng)
		if predictionOk && predictionTimeOk {
			metric += fmt.Sprintf("prediction_available=\"%v\",", true)
			metric += fmt.Sprintf("prediction_quality=\"%v\",", prediction.PredictionQuality)
			metric += fmt.Sprintf("prediction_tdiff=\"%v\",", time.Now().Unix()-predictionTime)
			metric += fmt.Sprintf("prediction_sgid=\"%v\",", prediction.SignalGroupId)
		} else {
			metric += fmt.Sprintf("prediction_available=\"%v\",", false)
			metric += fmt.Sprintf("prediction_quality=\"%v\",", -1)
			metric += fmt.Sprintf("prediction_tdiff=\"%v\",", 0)
			metric += fmt.Sprintf("prediction_sgid=\"%v\",", "")
		}
		metric += fmt.Sprintf("thing_name=\"%v\",", thing.Name)
		metric += fmt.Sprintf("thing_lanetype=\"%v\",", thing.Properties.LaneType)
		metric += "}"
		if predictionOk && predictionTimeOk {
			metric += fmt.Sprintf(" %v", prediction.PredictionQuality)
		} else {
			metric += " -1"
		}
		metrics = append(metrics, metric)
	}

	locationsGeoJson, err := locationFeatureCollection.MarshalJSON()
	if err != nil {
		log.Error.Println("Error marshalling geojson:", err)
		return
	}
	ioutil.WriteFile(staticPath+"predictions-locations.geojson", locationsGeoJson, 0644)

	lanesGeoJson, err := laneFeatureCollection.MarshalJSON()
	if err != nil {
		log.Error.Println("Error marshalling geojson:", err)
		return
	}
	ioutil.WriteFile(staticPath+"predictions-lanes.geojson", lanesGeoJson, 0644)

	// Write the metrics to a file.
	metricsString := ""
	for _, metric := range metrics {
		metricsString += metric + "\n"
	}
	// A txt file to directly display it in the browser without downloading it.
	ioutil.WriteFile(staticPath+"metrics.txt", []byte(metricsString), 0644)
}
