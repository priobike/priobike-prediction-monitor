package history

import (
	"encoding/json"
	"io/ioutil"
	"monitor/log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Response struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

// Periodically sync the history from our Prometheus.
func Sync() {
	for {
		log.Info.Println("Syncing history...")

		// Get the Prometheus api base url from the environment.
		baseUrl := os.Getenv("PROMETHEUS_URL")
		if baseUrl == "" {
			panic("PROMETHEUS_URL is not set")
		}

		prometheusExpressionListDayHistory := []string{
			// Number of all possible predictions.
			"prediction_service_subscription_count_total",
			// Number of published predictions.
			// For the day history we want a granularity of 30 minutes.
			// Each subscription is valid for 120 seconds.
			// Therefore we get, how many predictions were published in the last 1800 seconds (30 minutes)
			// and divide it by 15, to get how many predictions we published on average every 120 seconds.
			// The division by 2 is necessary, because TODO (copied from Grafana).
			"increase(prediction_service_predictions_count_total{}[1800s]) / 15 / 2",
		}

		dayHistory := make(map[string][]Result)

		currentTime := time.Now()

		// Fetch the day history. Get each of the metric for the last 24 hours (each 30 minutes).
		while := currentTime.Add(-24 * time.Hour)
		until := currentTime
		step := 30 * time.Minute
		for _, expression := range prometheusExpressionListDayHistory {
			// Fetch the data from Prometheus.
			prometheusResponse, err := http.Get(baseUrl + "/api/v1/query_range?query=" + url.QueryEscape(expression) + "&start=" + while.Format(time.RFC3339) + "&end=" + until.Format(time.RFC3339) + "&step=" + step.String())
			if err != nil {
				log.Warning.Println("Could not sync history:", err)
				break
			}
			defer prometheusResponse.Body.Close()

			body, err := ioutil.ReadAll(prometheusResponse.Body)
			if err != nil {
				log.Warning.Println("Could not sync history:", err)
				break
			}

			// Parse the response.
			var prometheusResponseParsed Response
			if err := json.Unmarshal(body, &prometheusResponseParsed); err != nil {
				log.Warning.Println("Could not sync history:", err)
				break
			}

			// Add the data to the history.
			for _, result := range prometheusResponseParsed.Data.Result {
				// Get the topic from the metric.
				topic := result.Metric.Topic

				// Get the values from the metric.
				for _, value := range result.Values {
					// Get the timestamp.
					timestamp := value[0].(float64)

					// Get the value.
					value := value[1].(string)

					// Add the value to the history.
					dayHistory[topic] = append(dayHistory[topic], Result{
						Metric: result.Metric,
						Value:  []interface{}{timestamp, value},
					})
				}
			}
		}

		// Sleep for 30 minutes.
		time.Sleep(30 * time.Minute)
	}
}
