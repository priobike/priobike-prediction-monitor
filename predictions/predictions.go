package predictions

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// The prediction model.
type Prediction struct {
	GreentimeThreshold int64   `json:"greentimeThreshold"`
	PredictionQuality  float64 `json:"predictionQuality"`
	SignalGroupId      string  `json:"signalGroupId"`
	StartTime          string  `json:"startTime"`
	Value              []int64 `json:"value"`
	Timestamp          string  `json:"timestamp"`
}

// Parse the timestamp to unix time.
func (p *Prediction) parseTimestamp() (int64, error) {
	timestamp := strings.ReplaceAll(p.StartTime, "[UTC]", "")
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// try to insert seconds
		parts := strings.Split(timestamp, "Z")
		timestamp = fmt.Sprintf("%s:00Z%s", parts[0], parts[1])
		parsed, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			fmt.Println(err)
			return 0, err
		}
	}
	unix := parsed.Unix()
	return unix, nil
}

// A map that contains the current prediction for each mqtt topic.
var Current = make(map[string]Prediction)

// A map that contains timestamps of the last prediction for each mqtt topic.
var Timestamps = make(map[string]int64)

// A callback that is executed when new messages arrive on the mqtt topic.
func onMessageReceived(client mqtt.Client, msg mqtt.Message) {
	// Parse the prediction from the message.
	var prediction Prediction
	if err := json.Unmarshal(msg.Payload(), &prediction); err != nil {
		fmt.Println(err)
		return
	}
	// Update the prediction for the connection.
	Current[msg.Topic()] = prediction
	// Update the timestamp for the connection with the current unix timestamp.
	unixtime, err := prediction.parseTimestamp()
	if err == nil {
		Timestamps[msg.Topic()] = unixtime
	}
}

// Listen for new predictions via mqtt.
func Listen() {
	fmt.Println("Starting prediction listener...")
	// Start a mqtt client that listens to all messages on the prediction
	// service mqtt. The mqtt broker is secured with a username and password.
	// The credentials and the mqtt url are loaded from environment variables.
	mqttUrl := os.Getenv("MQTT_URL")
	if mqttUrl == "" {
		panic("MQTT_URL not set")
	}
	mqttUsername := os.Getenv("MQTT_USERNAME")
	mqttPassword := os.Getenv("MQTT_PASSWORD")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttUrl)
	if mqttUsername != "" && mqttPassword != "" {
		opts.SetUsername(mqttUsername)
		opts.SetPassword(mqttPassword)
	}
	opts.SetClientID(fmt.Sprintf("priobike-prediction-monitor-%d", rand.Int()))
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		onMessageReceived(client, msg)
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := client.Subscribe("#", 2, nil); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Wait forever.
	select {}
}
