package sync

import (
	"encoding/json"
	"io/ioutil"
	"monitor/log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// A things response from the SensorThings API.
type ThingsResponse struct {
	Value   []Thing `json:"value"`
	NextUri *string `json:"@iot.nextLink"`
}

// A map that contains all things by their prediction mqtt topic.
var Things = make(map[string]Thing)

// Periodically sync the things from the SensorThings API.
func Run() {
	for {
		log.Info.Println("Syncing things...")

		// Get the SensorThings api base url from the environment.
		baseUrl := os.Getenv("SENSORTHINGS_URL")
		if baseUrl == "" {
			panic("SENSORTHINGS_URL is not set")
		}

		// Get the SensorThings query from the environment.
		query := os.Getenv("SENSORTHINGS_QUERY")
		if query == "" {
			panic("SENSORTHINGS_QUERY is not set")
		}

		// Fetch all pages of the SensorThings query.
		var pageUrl = baseUrl + "Things?%24filter=" + url.QueryEscape(query)
		for {
			resp, err := http.Get(pageUrl)
			if err != nil {
				log.Warning.Println("Could not sync things:", err)
				break
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Warning.Println("Could not sync things:", err)
				break
			}

			var thingsResponse ThingsResponse
			if err := json.Unmarshal(body, &thingsResponse); err != nil {
				log.Warning.Println("Could not sync things:", err)
				break
			}

			for _, thing := range thingsResponse.Value {
				// Validate that the thing has a location.
				_, _, err := thing.LatLng()
				if err != nil {
					log.Warning.Println("Could not get location for thing:", err)
					continue
				}
				Things[thing.Topic()] = thing
			}

			if thingsResponse.NextUri == nil {
				break
			}
			pageUrl = *thingsResponse.NextUri
		}

		log.Info.Printf("Synced %d things", len(Things))

		// Sleep for 1 hour.
		time.Sleep(1 * time.Hour)
	}
}
