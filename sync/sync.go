package sync

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
		fmt.Println("Syncing locations and other data of the things...")

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
				fmt.Println(err)
				break
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				break
			}

			var thingsResponse ThingsResponse
			if err := json.Unmarshal(body, &thingsResponse); err != nil {
				fmt.Println(err)
				break
			}

			for _, thing := range thingsResponse.Value {
				// Validate that the thing has a location.
				_, _, err := thing.LatLng()
				if err != nil {
					fmt.Printf("WARNING: Could not get location of thing %d: %s\n", thing.IotId, err)
					continue
				}
				Things[thing.Topic()] = thing
			}

			if thingsResponse.NextUri == nil {
				break
			}
			pageUrl = *thingsResponse.NextUri
		}

		fmt.Printf("Finished sync. Found %d things.\n", len(Things))

		// Sleep for 1 hour.
		time.Sleep(1 * time.Hour)
	}
}
