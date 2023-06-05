package history

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"monitor/log"
	"net/http"
	"net/url"
	"os"
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
		log.Info.Println("Syncing history...")

		prometheusExpressionListDayHistory := map[string]string{
			// Number of all possible predictions.
			"prediction_service_subscription_count_total": "prediction_service_subscription_count_total",
			// Number of published predictions.
			// For the day history we want a granularity of 30 minutes.
			// Each subscription is valid for 120 seconds.
			// Therefore we get, how many predictions were published in the last 1800 seconds (30 minutes)
			// and divide it by 15, to get how many predictions we published on average every 120 seconds.
			// The division by 2 is necessary, because TODO (copied from Grafana).
			"average_prediction_service_predictions_count_total": "increase(prediction_service_predictions_count_total{}[1800s]) / 15 / 2",
		}

		dayHistory := make(map[string][][]interface{})
		validDayHistory := true

		currentTime := time.Now()

		// Fetch the day history. Get each of the metric for the last 24 hours (each 30 minutes).
		while := currentTime.Add(-24 * time.Hour)
		until := currentTime
		step := 30 * time.Minute
		for key, expression := range prometheusExpressionListDayHistory {
			// Fetch the data from Prometheus.
			prometheusResponse, err := http.Post(
				baseUrl+"/api/v1/query_range",
				"application/x-www-form-urlencoded",
				bytes.NewBufferString(
					"query=("+url.QueryEscape(expression)+")&start="+while.Format(time.UnixDate)+"&end="+until.Format(time.UnixDate)+"&step="+step.String()))
			if err != nil {
				log.Warning.Println("Could not sync history:", err)
				validDayHistory = false
				break
			}
			defer prometheusResponse.Body.Close()

			body, err := ioutil.ReadAll(prometheusResponse.Body)
			if err != nil {
				log.Warning.Println("Could not sync history:", err)
				validDayHistory = false
				break
			}

			// Parse the response.
			var prometheusResponseParsed Response
			if err := json.Unmarshal(body, &prometheusResponseParsed); err != nil {
				log.Warning.Println("Could not sync history:", err)
				validDayHistory = false
				break
			}

			if prometheusResponseParsed.Warnings != nil {
				for _, warning := range prometheusResponseParsed.Warnings {
					log.Warning.Println("Warning got returned by Prometheus:", warning)
				}
			}

			if prometheusResponseParsed.Status != "success" {
				log.Warning.Println("Could not sync history:", prometheusResponseParsed.Status)
				log.Warning.Println("Error type:", prometheusResponseParsed.ErrorType)
				log.Warning.Println("Error:", prometheusResponseParsed.Error)
				validDayHistory = false
				break
			}

			if prometheusResponseParsed.Data.ResultType != "matrix" {
				log.Warning.Println("Could not sync history: ResultType is not matrix")
				validDayHistory = false
				break
			}

			// Add the data to the history.
			for _, result := range prometheusResponseParsed.Data.Result {
				dayHistory[key] = result.Values
			}
		}

		if validDayHistory {
			// Write the history update to the file.
			statusJson, err := json.Marshal(dayHistory)
			if err != nil {
				log.Error.Println("Error marshalling history summary:", err)
				validDayHistory = false
				return
			}
			ioutil.WriteFile(staticPath+"day-history.json", statusJson, 0644)
		}

		// Sleep for 30 minutes.
		time.Sleep(30 * time.Minute)
	}
}
