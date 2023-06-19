package history

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"monitor/log"
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

type Metric struct {
	Name     string `json:"__name__"`
	Instance string `json:"instance"`
	Job      string `json:"job"`
}

// Creates the history.
func createHistory(baseUrl string, staticPath string, forHoursInPast int, intervalMinutes int, name string) {
	validHistory := true

	log.Info.Println("Syncing ", name, " history...")

	intervalSeconds := intervalMinutes * 60
	intervalFactor := intervalSeconds / 120 // To get the average of published predictions each 120 seconds.

	prometheusExpressionListHistory := map[string]string{
		// Number of all possible predictions.
		// "OR vector(0)" is necessary, because otherwise if Prometheus has no data for the given time range,
		// it will return no data instead of a zero value.
		"prediction_service_subscription_count_total": "prediction_service_subscription_count_total OR vector(0)",
		// Number of published predictions.
		// "OR vector(0)" is necessary, because otherwise if Prometheus has no data for the given time range,
		// it will return no data instead of a zero value.
		"average_prediction_service_predictions_count_total": "increase(prediction_service_predictions_count_total{}[" + strconv.Itoa(intervalSeconds) + "s]) / " + strconv.Itoa(intervalFactor) + " / 2 OR vector(0)",
	}

	historyWithList := make(map[string][][]interface{})
	historyWithMap := make(map[string]map[int]float64)

	currentTime := time.Now()

	// Fetch the history.
	while := currentTime.Add(time.Hour * -time.Duration(forHoursInPast))
	until := currentTime
	step := time.Duration(intervalMinutes) * time.Minute
	for key, expression := range prometheusExpressionListHistory {
		// Fetch the data from Prometheus.
		// Example response:
		// {"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1685888801,"0"],[1685890601,"0"],[1685892401,"0"],[1685894201,"0"],[1685896001,"0"],[1685897801,"0"],[1685899601,"0"],[1685901401,"0"],[1685903201,"0"],[1685905001,"0"],[1685906801,"0"],[1685908601,"0"],[1685910401,"0"],[1685912201,"0"],[1685914001,"0"],[1685915801,"0"],[1685917601,"0"],[1685919401,"0"],[1685921201,"0"],[1685923001,"0"],[1685924801,"0"],[1685926601,"0"],[1685928401,"0"],[1685930201,"0"],[1685932001,"0"],[1685933801,"0"],[1685935601,"0"],[1685937401,"0"],[1685939201,"0"],[1685941001,"0"],[1685942801,"0"],[1685944601,"0"],[1685946401,"0"],[1685948201,"0"],[1685950001,"0"],[1685951801,"0"],[1685953601,"0"],[1685955401,"0"],[1685957201,"0"],[1685959001,"0"],[1685960801,"0"],[1685962601,"0"],[1685964401,"0"],[1685966201,"0"],[1685968001,"0"],[1685969801,"0"],[1685971601,"0"],[1685973401,"0"],[1685975201,"0"]]},{"metric":{"__name__":"prediction_service_subscription_count_total","instance":"prediction-service:8000","job":"staging-prediction-service"},"values":[[1685966201,"2"],[1685968001,"2"],[1685969801,"2"],[1685971601,"2"],[1685973401,"2"]]}]}}
		prometheusResponse, err := http.Post(
			baseUrl+"/api/v1/query_range",
			"application/x-www-form-urlencoded",
			bytes.NewBufferString(
				"query=("+url.QueryEscape(expression)+")&start="+strconv.FormatInt(while.Unix(), 10)+"&end="+strconv.FormatInt(until.Unix(), 10)+"&step="+step.String()))
		if err != nil {
			log.Warning.Println("Could not sync ", name, " history:", err)
			validHistory = false
			break
		}
		defer prometheusResponse.Body.Close()

		body, err := ioutil.ReadAll(prometheusResponse.Body)
		if err != nil {
			log.Warning.Println("Could not sync ", name, " history:", err)
			validHistory = false
			break
		}

		// Parse the response.
		var prometheusResponseParsed Response
		if err := json.Unmarshal(body, &prometheusResponseParsed); err != nil {
			log.Warning.Println("Could not sync ", name, " history:", err)
			validHistory = false
			break
		}

		if prometheusResponseParsed.Warnings != nil {
			for _, warning := range prometheusResponseParsed.Warnings {
				log.Warning.Println("Warning got returned by Prometheus (", name, " history):", warning)
			}
		}

		if prometheusResponseParsed.Status != "success" {
			log.Warning.Println("Could not sync ", name, " history:", prometheusResponseParsed.Status)
			log.Warning.Println("Error type:", prometheusResponseParsed.ErrorType)
			log.Warning.Println("Error:", prometheusResponseParsed.Error)
			validHistory = false
			break
		}

		if prometheusResponseParsed.Data.ResultType != "matrix" {
			log.Warning.Println("Could not sync ", name, " history: ResultType is not matrix")
			validHistory = false
			break
		}

		// Add results where the value for every timestamp is "0"
		historyWithList[key] = prometheusResponseParsed.Data.Result[0].Values

		// Overwrite the values of timestamps where we have actual data.
		for _, value := range prometheusResponseParsed.Data.Result[1].Values {
			for i, existingValue := range historyWithList[key] {
				if existingValue[0] == value[0] {
					historyWithList[key][i] = value
					break
				}
			}
		}

		// If we have less than 48 values, we have a gap in the data.
		if len(historyWithList[key]) < 48 {
			log.Warning.Println("Something went wrong while syncing ", name, " history: We have less than 48 values for ", key)
		}

		// Sort the history by time.
		sort.Slice(historyWithList[key], func(i, j int) bool {
			return historyWithList[key][i][0].(float64) < historyWithList[key][j][0].(float64)
		})

		// Convert to map.
		for _, value := range historyWithList[key] {
			float, err := strconv.ParseFloat(value[1].(string), 64)
			if err != nil {
				log.Warning.Println("During history sync a prediction is not of type float64: ", value[1].(string))
				continue
			}
			if historyWithMap[key] == nil {
				historyWithMap[key] = make(map[int]float64)
			}
			historyWithMap[key][int(value[0].(float64))] = float
		}
	}

	if validHistory {
		// Write the history update to the file.
		statusJson, err := json.Marshal(historyWithMap)
		if err != nil {
			log.Error.Println("Error marshalling ", name, " history summary:", err)
			validHistory = false
		}
		if validHistory {
			ioutil.WriteFile(staticPath+name+"-history.json", statusJson, 0644)
			log.Info.Println("Synced ", name, " history")
		}
	}
}

// Periodically sync the history from our Prometheus.
func Sync() {
	// Fetch the path under which we will save the json files.
	staticPath := os.Getenv("STATIC_PATH")
	if staticPath == "" {
		panic("STATIC_PATH not set")
	}

	// Get the Prometheus api base url from the environment.
	baseUrl := os.Getenv("PROMETHEUS_URL")
	if baseUrl == "" {
		panic("PROMETHEUS_URL is not set")
	}

	for {
		// Sync the day history.
		createHistory(baseUrl, staticPath, 24, 30, "day")

		// Sync the week history.
		createHistory(baseUrl, staticPath, 168, 30, "week")

		// Sleep for 1 minute.
		time.Sleep(1 * time.Minute)
	}
}
