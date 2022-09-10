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

// Get the location of a thing. This is the first coordinate of the ingress lane of the thing.
func (thing Thing) LatLng() (*float64, *float64, error) {
	if len(thing.Locations) == 0 {
		return nil, nil, fmt.Errorf("thing has no locations")
	}
	lanes := thing.Locations[0].Location.Geometry.Coordinates
	if len(lanes) < 2 {
		return nil, nil, fmt.Errorf("thing has no ingress lane")
	}
	ingressLane := lanes[1]
	if len(ingressLane) < 1 {
		return nil, nil, fmt.Errorf("ingress lane has no coordinates")
	}
	coordinate := ingressLane[0]
	return &coordinate[1], &coordinate[0], nil
}

// Get the mqtt topic of a thing. This is `hamburg`/name.
func (thing Thing) Topic() string {
	return fmt.Sprintf("hamburg/%s", thing.Name)
}
