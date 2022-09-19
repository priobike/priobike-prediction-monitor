package status

import (
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
}
