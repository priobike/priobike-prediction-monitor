package main // TODO: change back

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"
)

type Response struct {
	Status    string   `json:"status"`
	Data      Data     `json:"data"`
	ErrorType string   `json:"errorType,omitempty"`
	Error     string   `json:"error,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric Metric          `json:"metric"`
	Values [][]interface{} `json:"values"`
}

type DataPoint struct {
	Timestamp int
	Value     float64
}

type Metric struct {
	Name     string `json:"__name__"`
	Instance string `json:"instance"`
	Job      string `json:"job"`
	Le       string `json:"le"`
}

// Set to true to use a local json file instead of fetching the data from Prometheus.
// Debug-Features:
var debugMode = true                          // prevents the script from running in an infinite loop
var debugLoadFromFileInsteadOfFetching = true // loads the data from a json file instead of fetching it from Prometheus, naming convention: debug_<key>.json
var debugSaveFetchesFileToDisk = true         // saves the data from Prometheus to a json file, if the service is reachable from the script
var debugSaveFinishedListToDisk = true        // saves the good predictions to a json file

// Create the history.
func createHistory(staticPath string, forHoursInPast int, intervalMinutes int, name string) {

	println("Syncing ", name, " history...")

	intervalSeconds := intervalMinutes * 60
	intervalFactor := intervalSeconds / 120 // To get the average of published predictions each 120 seconds.

	historyEncodedAsMap := make(map[string]map[int]float64)

	// Number of published predictions with prediction quality.
	// "OR vector(0)" is necessary, because otherwise if Prometheus has no data for the given time range,
	// it will return no data instead of a zero value.
	// In summary.go bad prediction quality is defined as <= 50.0, therefore we need at least 60.0
	key := "prediction_service_prediction_quality_distribution_bucket"
	// +Inf contains everything that is smaller than infinity (so everything) and then we subtract the bad predictions, ie. everything that is <= 50.0
	part1 := "sum(increase(prediction_service_prediction_quality_distribution_bucket{le=\"+Inf\"}[1800s]) / 15 / 2)-"
	part2 := "sum(increase(prediction_service_prediction_quality_distribution_bucket{le=\"50.0\"}[1800s]) / 15 / 2)"
	expression := part1 + part2
	println("Debug Expression: ", expression)
	processedData, validHistory := processResponse(key, expression, staticPath, forHoursInPast, intervalMinutes, name)

	// Add list with key to map
	if validHistory {
		historyEncodedAsMap[key] = make(map[int]float64)
		for _, value := range processedData {
			// if key already exists, add the value to the existing value
			historyEncodedAsMap[key][value.Timestamp] += value.Value
		}
	}

	// Number of all possible predictions.
	// "OR vector(0)" is necessary, because otherwise if Prometheus has no data for the given time range,
	// it will return no data instead of a zero value.
	key = "prediction_service_subscription_count_total"
	expression = "prediction_service_subscription_count_total OR vector(0)"
	processedData, validHistory = processResponse(key, expression, staticPath, forHoursInPast, intervalMinutes, name)

	// Add list with key to map
	if validHistory {
		historyEncodedAsMap[key] = make(map[int]float64)
		for _, value := range processedData {
			historyEncodedAsMap[key][value.Timestamp] = value.Value
		}
	}

	// Number of published predictions.
	// "OR vector(0)" is necessary, because otherwise if Prometheus has no data for the given time range,
	// it will return no data instead of a zero value.
	key = "average_prediction_service_predictions_count_total"
	expression = "increase(prediction_service_predictions_count_total{}[" + strconv.Itoa(intervalSeconds) + "s]) / " + strconv.Itoa(intervalFactor) + " / 2 OR vector(0)"
	processedData, validHistory = processResponse(key, expression, staticPath, forHoursInPast, intervalMinutes, name)

	// Add list with key to map
	if validHistory {
		historyEncodedAsMap[key] = make(map[int]float64)
		for _, value := range processedData {
			historyEncodedAsMap[key][value.Timestamp] = value.Value
		}
	}

	// if historyEncodesAsMap is not empty, write it to the file
	if len(historyEncodedAsMap) > 0 {
		// Write the history update to the file.
		statusJson, err := json.Marshal(historyEncodedAsMap)
		if err != nil {
			println("Error marshalling ", name, " history summary:", err)
			validHistory = false
		}
		if validHistory {
			os.WriteFile(staticPath+name+"-history.json", statusJson, 0644)
			println("Synced ", name, " history")
		}
	}
}

// Fetches the data from Prometheus. It is usually not reachable from outside of the VM.
func fetchFromPrometheus(staticPath string, forHoursInPast int, intervalMinutes int, name string, expression string) (prometheusResponseParsed Response, err error, validHistory bool) {

	// TODO: remove and use static path
	baseUrl := "https://priobike.vkw.tu-dresden.de/staging/prometheus"
	baseUrl = "http://prometheus:9090"

	//currentTime := time.Now()

	// Fetch the history.
	// while := currentTime.Add(time.Hour * -time.Duration(forHoursInPast))
	// until := currentTime
	//step := time.Duration(intervalMinutes) * time.Minute

	validHistory = true

	// Fetch the data from Prometheus.
	// Example response:
	// {"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1685888801,"0"],[1685890601,"0"],[1685892401,"0"],[1685894201,"0"],[1685896001,"0"],[1685897801,"0"],[1685899601,"0"],[1685901401,"0"],[1685903201,"0"],[1685905001,"0"],[1685906801,"0"],[1685908601,"0"],[1685910401,"0"],[1685912201,"0"],[1685914001,"0"],[1685915801,"0"],[1685917601,"0"],[1685919401,"0"],[1685921201,"0"],[1685923001,"0"],[1685924801,"0"],[1685926601,"0"],[1685928401,"0"],[1685930201,"0"],[1685932001,"0"],[1685933801,"0"],[1685935601,"0"],[1685937401,"0"],[1685939201,"0"],[1685941001,"0"],[1685942801,"0"],[1685944601,"0"],[1685946401,"0"],[1685948201,"0"],[1685950001,"0"],[1685951801,"0"],[1685953601,"0"],[1685955401,"0"],[1685957201,"0"],[1685959001,"0"],[1685960801,"0"],[1685962601,"0"],[1685964401,"0"],[1685966201,"0"],[1685968001,"0"],[1685969801,"0"],[1685971601,"0"],[1685973401,"0"],[1685975201,"0"]]},{"metric":{"__name__":"prediction_service_subscription_count_total","instance":"prediction-service:8000","job":"staging-prediction-service"},"values":[[1685966201,"2"],[1685968001,"2"],[1685969801,"2"],[1685971601,"2"],[1685973401,"2"]]}]}}
	urlRequest := baseUrl + "/api/v1/query_range"
	contentTypeRequest := "application/x-www-form-urlencoded"
	//bodyRequest :=		"query=(" + url.QueryEscape(expression) + ")&start=" + strconv.FormatInt(while.Unix(), 10) + "&end=" + strconv.FormatInt(until.Unix(), 10) + "&step=" + step.String()
	bodyRequest :=
		"query=(" + url.QueryEscape(expression) + ")&start=1698649200&end=1698656400&step=15s"
		// TODO: hardcode start und end

	println("Debug Query: curl -d \"" + bodyRequest + "\" -X POST " + urlRequest)

	prometheusResponse, err := http.Post(urlRequest, contentTypeRequest, bytes.NewBufferString(bodyRequest))
	if err != nil {
		println("Could not sync ", name, " history:", err)
		validHistory = false
	}

	defer prometheusResponse.Body.Close()

	body, err := io.ReadAll(prometheusResponse.Body)
	if err != nil {
		println("Could not sync ", name, " history:", err)
		validHistory = false
	}

	// Parse the response.
	if err := json.Unmarshal(body, &prometheusResponseParsed); err != nil {
		println("Could not sync ", name, " history:", err)
		validHistory = false
	}
	return prometheusResponseParsed, err, validHistory
}

// Loads a json file for debugging purposes.
func loadFromFile(key string) (result Response, err error, validHistory bool) {
	jsonRaw, err := os.ReadFile("debug_" + key + ".json")

	if err != nil {
		println("Could not read file:", err)
	}
	json.Unmarshal(jsonRaw, &result)
	return result, err, true
}

func processResponse(key string, expression string, staticPath string, forHoursInPast int, intervalMinutes int, name string) (finishedList []DataPoint, validHistory bool) {

	validHistory = true

	var history Response
	var err error
	if !debugLoadFromFileInsteadOfFetching {
		history, err, validHistory = fetchFromPrometheus(staticPath, forHoursInPast, intervalMinutes, name, expression)
		if err != nil {
			return finishedList, validHistory
		}

		if debugSaveFetchesFileToDisk {
			// Save to json file
			statusJson, err := json.Marshal(history)
			if err != nil {
				println("Error marshalling ", name, " history summary:", err)
			}

			os.WriteFile("debug_"+key+".json", statusJson, 0644)
		}

	} else {
		// Debug mode.
		history, err, validHistory = loadFromFile(key)
		if err != nil {
			return finishedList, validHistory
		}
	}

	if history.Warnings != nil {
		for _, warning := range history.Warnings {
			println("Warning got returned by Prometheus (", name, " history):", warning)
		}
	}

	if history.Status != "success" {
		println("Could not sync ", name, " history:", history.Status)
		println("Error type:", history.ErrorType)
		println("Error:", history.Error)
		validHistory = false
		return finishedList, validHistory
	}

	if history.Data.ResultType != "matrix" {
		println("Could not sync ", name, " history: ResultType is not matrix")
		validHistory = false
		return finishedList, validHistory
	}

	if len(history.Data.Result[0].Values) < 48 {
		println("Something went wrong while syncing ", name, " history: We have less than 48 values for ", key)
	}

	// Convert to list of DataPoints
	for _, valuePack := range history.Data.Result {
		for _, datapoints := range valuePack.Values {
			timestamp := int(datapoints[0].(float64))
			value, err := strconv.ParseFloat(datapoints[1].(string), 64)
			if err != nil {
				println("During history sync a prediction is not of type float64: ", datapoints[1].(string))
				continue
			}
			finishedList = append(finishedList, DataPoint{Timestamp: timestamp, Value: value})
		}
	}

	// Sort by the timestamp.
	sort.Slice(finishedList, func(i, j int) bool {
		return finishedList[i].Timestamp < finishedList[j].Timestamp
	})

	if debugSaveFinishedListToDisk {
		// Save to json file
		statusJson, err := json.Marshal(finishedList)
		if err != nil {
			println("Error marshalling ", name, " history summary:", err)
		}
		os.WriteFile("FinishedList_"+key+".json", statusJson, 0644)
	}

	return finishedList, validHistory
}

// Periodically sync the history from our Prometheus.
// func Sync() {
// 	// Fetch the path under which we will save the json files.
// 	staticPath := os.Getenv("STATIC_PATH")
// 	if staticPath == "" {
// 		panic("STATIC_PATH not set")
// 	}

// 	// Get the Prometheus api base url from the environment.
// 	baseUrl := os.Getenv("PROMETHEUS_URL")
// 	if baseUrl == "" {
// 		println("PROMETHEUS_URL is not set, history will not be synced.")
// 		return
// 	}

func main() {

	// TODO: change back
	var staticPath = ""

	if debugMode {
		// Sync the day history.
		createHistory(staticPath, 24, 30, "day")

		// Sync the week history.
		createHistory(staticPath, 168, 120, "week")
	} else {
		for {
			// Sync the day history.
			createHistory(staticPath, 24, 30, "day")

			// Sync the week history.
			createHistory(staticPath, 168, 120, "week")

			// Sleep for 1 minute.
			time.Sleep(1 * time.Minute)
		}
	}
}
