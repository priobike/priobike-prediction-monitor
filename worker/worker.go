package worker

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"monitor/log"
	"net/http"
	"os"
	"time"
)

func fetchFile(url string, path string) {
	// Create the path
	out, err := os.Create(path)
	if err != nil {
		panic("Could not create file: " + path + " Error: " + err.Error())
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		panic("Could not fetch file: " + url + " Error: " + err.Error())
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		panic("Could not fetch file, bad return status: " + url + " Error: " + resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		panic("Could not write to file: " + path + " Error: " + err.Error())
	}
}

func fetchPredictionsLocations(staticPath string, managerStaticURL string) {
	fetchFile(managerStaticURL+"/predictions-locations.geojson", staticPath+"/predictions-locations.geojson")
	log.Info.Println("Fetched predictions-locations.geojson")
}

func fetchPredictionsLanes(staticPath string, managerStaticURL string) {
	fetchFile(managerStaticURL+"/predictions-lanes.geojson", staticPath+"/predictions-lanes.geojson")
	log.Info.Println("Fetched predictions-lanes.geojson")
}

func fetchSummary(staticPath string, managerStaticURL string) {
	fetchFile(managerStaticURL+"/status.json", staticPath+"/status.json")
	log.Info.Println("Fetched status.json")
}

func fetchSGStatus(staticPath string, managerStaticURL string) {
	var thingIndex []string

	resp, getErr := http.Get(managerStaticURL + "/thing-index.json")
	if getErr != nil {
		panic("Could not fetch file: " + managerStaticURL + "/thing-index.json" + " Error: " + getErr.Error())
	}

	respBody, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		panic("Could not read file: " + managerStaticURL + "/thing-index.json" + " Error: " + readErr.Error())
	}

	unmarshalErr := json.Unmarshal(respBody, &thingIndex)
	if unmarshalErr != nil {
		panic("Could not unmarshal file: " + managerStaticURL + "/thing-index.json" + " Error: " + unmarshalErr.Error())
	}

	for _, topic := range thingIndex {
		// Create the thing directory if it does not exist
		if _, err := os.Stat(staticPath + "/" + topic); os.IsNotExist(err) {
			mkDirErr := os.MkdirAll(staticPath+"/"+topic, 0755)
			if mkDirErr != nil {
				panic("Could not create directory: " + staticPath + "/" + topic + " Error: " + mkDirErr.Error())
			}
		}

		fetchFile(managerStaticURL+"/"+topic+"/status.json", staticPath+"/"+topic+"/status.json")
	}
	log.Info.Println("Fetched SG status files")
}

func Run() {
	staticPath := os.Getenv("STATIC_PATH")
	if staticPath == "" {
		panic("STATIC_PATH not set")
	}

	managerStaticURL := os.Getenv("MANAGER_STATIC_URL")
	if managerStaticURL == "" {
		panic("MANAGER_STATIC_URL not set")
	}

	for {
		log.Info.Println("Start file sync...")
		fetchPredictionsLocations(staticPath, managerStaticURL)
		fetchPredictionsLanes(staticPath, managerStaticURL)
		fetchSummary(staticPath, managerStaticURL)
		fetchSGStatus(staticPath, managerStaticURL)
		log.Info.Println("File sync done.")
		// Wait random time between 40 and 90 seconds to not overload the server
		waitTime := time.Duration(40+(rand.Intn(50))) * time.Second
		time.Sleep(waitTime)
	}
}
