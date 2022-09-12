package sync

import "fmt"

// A thing from the SensorThings API.
type Thing struct {
	Description string `json:"description"`
	IotId       int    `json:"@iot.id"`
	Name        string `json:"name"`
	Properties  struct {
		Topic           string   `json:"topic"`
		AssetID         string   `json:"assetID"`
		Keywords        []string `json:"keywords"`
		LaneType        string   `json:"laneType"`
		Language        string   `json:"language"`
		OwnerThing      string   `json:"ownerThing"`
		ConnectionID    string   `json:"connectionID"`
		EgressLaneID    string   `json:"egressLaneID"`
		IngressLaneID   string   `json:"ingressLaneID"`
		InfoLastUpdated string   `json:"infoLastUpdated"`
		TrafficLightsID string   `json:"trafficLightsID"`
	} `json:"properties"`
	SelfLink  string     `json:"@iot.selfLink"`
	Locations []Location `json:"Locations"`
}

// Get the lane of a thing. This is the connection lane of the thing.
func (thing Thing) Lane() ([][]float64, error) {
	if len(thing.Locations) == 0 {
		return nil, fmt.Errorf("thing %s has no locations", thing.Name)
	}
	lanes := thing.Locations[0].Location.Geometry.Coordinates
	if len(lanes) < 2 {
		return nil, fmt.Errorf("thing %s has no ingress lane", thing.Name)
	}
	connectionLane := lanes[1] // 0: ingress lane, 1: connection lane, 2: egress lane
	if len(connectionLane) < 1 {
		return nil, fmt.Errorf("connection lane has no coordinates for thing %s", thing.Name)
	}
	return connectionLane, nil
}

// Get the mqtt topic of a thing. This is `hamburg`/name.
func (thing Thing) Topic() string {
	return fmt.Sprintf("hamburg/%s", thing.Name)
}
