package history

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"monitor/log"
	"net/http"
	"net/url"
	"os"
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

	validDayHistory := true

	for {
		log.Info.Println("Syncing history...")

		prometheusExpressionListDayHistory := map[string]string{
			// Number of all possible predictions.
			// "OR vector(0)" is necessary, because otherwise if Prometheus has no data for the given time range,
			// it will return no data instead of a zero value.
			"prediction_service_subscription_count_total": "prediction_service_subscription_count_total OR vector(0)",
			// Number of published predictions.
			// For the day history we want a granularity of 30 minutes.
			// Each subscription is valid for 120 seconds.
			// Therefore we get, how many predictions were published in the last 1800 seconds (30 minutes)
			// and divide it by 15, to get how many predictions we published on average every 120 seconds.
			// The division by 2 is necessary, because TODO (copied from Grafana).
			// "OR vector(0)" is necessary, because otherwise if Prometheus has no data for the given time range,
			// it will return no data instead of a zero value.
			"average_prediction_service_predictions_count_total": "increase(prediction_service_predictions_count_total{}[1800s]) / 15 / 2 OR vector(0)",
		}

		dayHistory := make(map[string][][]interface{})
		validDayHistory = true

		currentTime := time.Now()

		// Fetch the day history. Get each of the metric for the last 24 hours (each 30 minutes).
		while := currentTime.Add(-24 * time.Hour)
		until := currentTime
		step := 30 * time.Minute
		for key, expression := range prometheusExpressionListDayHistory {
			// Fetch the data from Prometheus.
			// Example response:
			// {"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1685888801,"0"],[1685890601,"0"],[1685892401,"0"],[1685894201,"0"],[1685896001,"0"],[1685897801,"0"],[1685899601,"0"],[1685901401,"0"],[1685903201,"0"],[1685905001,"0"],[1685906801,"0"],[1685908601,"0"],[1685910401,"0"],[1685912201,"0"],[1685914001,"0"],[1685915801,"0"],[1685917601,"0"],[1685919401,"0"],[1685921201,"0"],[1685923001,"0"],[1685924801,"0"],[1685926601,"0"],[1685928401,"0"],[1685930201,"0"],[1685932001,"0"],[1685933801,"0"],[1685935601,"0"],[1685937401,"0"],[1685939201,"0"],[1685941001,"0"],[1685942801,"0"],[1685944601,"0"],[1685946401,"0"],[1685948201,"0"],[1685950001,"0"],[1685951801,"0"],[1685953601,"0"],[1685955401,"0"],[1685957201,"0"],[1685959001,"0"],[1685960801,"0"],[1685962601,"0"],[1685964401,"0"],[1685966201,"0"],[1685968001,"0"],[1685969801,"0"],[1685971601,"0"],[1685973401,"0"],[1685975201,"0"]]},{"metric":{"__name__":"prediction_service_subscription_count_total","instance":"prediction-service:8000","job":"staging-prediction-service"},"values":[[1685966201,"2"],[1685968001,"2"],[1685969801,"2"],[1685971601,"2"],[1685973401,"2"]]}]}}
			prometheusResponse, err := http.Post(
				baseUrl+"/api/v1/query_range",
				"application/x-www-form-urlencoded",
				bytes.NewBufferString(
					"query=("+url.QueryEscape(expression)+")&start="+strconv.FormatInt(while.Unix(), 10)+"&end="+strconv.FormatInt(until.Unix(), 10)+"&step="+step.String()))
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
			// We get two results. One where each timestamp is "0" and one where for each timestamp where we have data, the value is not "0".
			// We want to merge these two results such that we always use the object with != "0" if available.
			for _, result := range prometheusResponseParsed.Data.Result {
				if dayHistory[key] == nil {
					dayHistory[key] = result.Values
				} else {
					indicesToUpdate := make([]int, 0)
					for _, value := range result.Values {
						for i, existingValue := range dayHistory[key] {
							if existingValue[0] == value[0] && value[1] != "0" {
								indicesToUpdate = append(indicesToUpdate, i)
							} else {
								indicesToUpdate = append(indicesToUpdate, -i)
							}
						}
					}
					for i, index := range indicesToUpdate {
						if index > 0 {
							dayHistory[key][index] = result.Values[i]
						}
					}
				}
			}

			// If we have less than 48 values, we have a gap in the data.
			if len(dayHistory[key]) < 48 {
				log.Warning.Println("Something went wrong while syncing history: We have less than 48 values for ", key)
			}
		}

		if validDayHistory {
			// Write the history update to the file.
			statusJson, err := json.Marshal(dayHistory)
			if err != nil {
				log.Error.Println("Error marshalling history summary:", err)
				validDayHistory = false
			}
			if validDayHistory {
				ioutil.WriteFile(staticPath+"day-history.json", statusJson, 0644)
				log.Info.Println("Synced history")
			}
		}

		// Sleep for 30 minutes.
		time.Sleep(30 * time.Minute)
	}
}
