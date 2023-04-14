package predictions

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"monitor/log"
	"os"
	"strings"
	"sync"
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
		if len(parts) != 2 {
			log.Warning.Println("Could not parse timestamp:", err)
			return 0, err
		}
		timestamp = fmt.Sprintf("%s:00Z%s", parts[0], parts[1])
		parsed, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			log.Warning.Println("Could not parse timestamp:", err)
			return 0, err
		}
	}
	unix := parsed.Unix()
	return unix, nil
}

// A mutex that protects writes to the current map.
var CurrentMutex = &sync.Mutex{}

// A map that contains the current prediction for each mqtt topic.
var Current = make(map[string]Prediction)

// A mutex that protects writes to the timestamps map.
var TimestampsMutex = &sync.Mutex{}

// A map that contains timestamps of the last prediction for each mqtt topic.
var Timestamps = make(map[string]int64)

// An integer that represents the number of messages received.
var received = 0

// A callback that is executed when new messages arrive on the mqtt topic.
func onMessageReceived(client mqtt.Client, msg mqtt.Message) {
	// Parse the prediction from the message.
	var prediction Prediction
	if err := json.Unmarshal(msg.Payload(), &prediction); err != nil {
		log.Warning.Println("Could not parse prediction:", err)
		return
	}
	// Update the prediction for the connection.
	CurrentMutex.Lock()
	Current[msg.Topic()] = prediction
	CurrentMutex.Unlock()
	// Update the timestamp for the connection with the current unix timestamp.
	unixtime, err := prediction.parseTimestamp()
	if err == nil {
		TimestampsMutex.Lock()
		Timestamps[msg.Topic()] = unixtime
		TimestampsMutex.Unlock()
	}
	// Increment the number of received messages.
	received++
}

// Print out the number of received messages periodically.
func Print() {
	for {
		time.Sleep(60 * time.Second)
		log.Info.Printf("Received %d predictions since service startup.", received)
	}
}

// Listen for new predictions via mqtt.
func Listen() {
	// Start a mqtt client that listens to all messages on the prediction
	// service mqtt. The mqtt broker is secured with a username and password.
	// The credentials and the mqtt url are loaded from environment variables.
	mqttUrl := os.Getenv("MQTT_URL")
	if mqttUrl == "" {
		panic("MQTT_URL not set")
	}
	log.Info.Println("Connecting to prediction mqtt broker at :", mqttUrl)

	mqttUsername := os.Getenv("MQTT_USERNAME")
	mqttPassword := os.Getenv("MQTT_PASSWORD")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttUrl)
	if mqttUsername != "" && mqttPassword != "" {
		opts.SetUsername(mqttUsername)
		opts.SetPassword(mqttPassword)
	}
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Info.Println("Connected to prediction mqtt broker.")
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Warning.Println("Connection to prediction mqtt broker lost:", err)
		panic("Intentionally crashing. The docker setup should handle a restart such that a new connection to the mqtt broker can be established.")
	})
	randSource := rand.NewSource(time.Now().UnixNano())
	random := rand.New(randSource)
	clientID := fmt.Sprintf("priobike-prediction-monitor-%d", random.Int())
	log.Info.Println("Using client id:", clientID)
	opts.SetClientID(clientID)
	opts.SetOrderMatters(false)
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Warning.Println("Received unexpected message on topic:", msg.Topic())
	})

	client := mqtt.NewClient(opts)
	if conn := client.Connect(); conn.Wait() && conn.Error() != nil {
		panic(conn.Error())
	}

	if sub := client.Subscribe("#", 1, onMessageReceived); sub.Wait() && sub.Error() != nil {
		panic(sub.Error())
	}

	// Print the number of received messages periodically.
	go Print()

	// Wait forever.
	select {}
}
