# priobike-prediction-monitor

This service can be used to monitor predictions of a traffic light prediction service. This service in itself does not make predictions, but you can configure it to listen to predictions made by a prediction service and monitor the predictions. It creates Prometheus metrics as well as a .json file for each traffic light with relevant information about the current prediction quality status of the traffic light. Furthermore it also creates .geojson file of all traffic light positions and their corresponding lane.

In the PrioBike project it monitors the predictions created by the [prediction services](https://github.com/priobike/priobike-prediction-service). We use the monitoring of the predictions in the app to provide information about the prediction quality along the routes of the users as well as to provide transparency to the users in case of data outages or other issues.

[Learn more about PrioBike](https://github.com/priobike)

## Important to know

This service can run in two modes: manager and worker. The worker mode is designed to face user traffic and can be scaled horizontally.

These are the exact tasks of each role:

- The worker is a simple NGINX web server. It receives the .geojson/.json files from the manager and serves them to the user. This service is stateless. After restarting, it will have no data until the manager sends it the data the first time.
- The manager subscribes to the MQTT borker where the predictions are published and periodically creates the metrics. After creation of the .geojson/.json files, it sends them to all workers. This is done every minute.

See docker-compose.yml for an example setup.

## Quickstart

Run using the provided docker-compose file:
```bash
docker-compose up
```

### Configuration

The service is configured using environment variables. The following variables are available:

#### Manager

- `MQTT_URL` The URL of the MQTT broker.
- `MQTT_PASSWORD` The password for the MQTT broker.
- `MQTT_USERNAME` The username for the MQTT broker.
- `SENSORTHINGS_URL` The URL of the SensorThings API (used to fetch information about the traffic lights).
- `SENSORTHINGS_QUERY` The query to fetch the relevant traffic lights from the SensorThings API.
- `STATIC_PATH` The path under which all resources will be stored for the web API. NOTE: The path must be provided with a trailing slash.
- `WORKER_HOST` The host of the worker. Required for the manager to send the .geojson/.json files to the worker.
- `WORKER_PORT` The port of the worker. Required for the manager to send the .geojson/.json files to the worker.
- `WORKER_BASIC_AUTH_USER` The username for the basic auth of the worker.
- `WORKER_BASIC_AUTH_PASS` The password for the basic auth of the worker.

#### Worker

- `BASIC_AUTH_USER` The username for the basic auth.
- `BASIC_AUTH_PASS` The password for the basic auth.

We use basic auth such that only the authorized manager can update the worker with the .geojson/.json files.

## API

Theoretically, the worker exposes all files sent to him. Since only the manager can send files and we know what files he is sending, the following endpoints/files are available under normal operation:

- `/predictions-lanes.geojson` The geojson file containing all traffic lights and their lanes.
- `/predictions-locations.geojson` The geojson file containing all traffic lights and their locations.
- `<ID>/status.json` The json file containing the status of the prediction quality of the traffic light with the given ID.

The manager exposes the Prometheus metrics at `/metrics.txt`.

## Contributing

We highly encourage you to open an issue or a pull request. You can also use our repository freely with the `MIT` license.

Every service runs through testing before it is deployed in our release setup. Read more in our [PrioBike deployment readme](https://github.com/priobike/.github/blob/main/wiki/deployment.md) to understand how specific branches/tags are deployed.

## Anything unclear?

Help us improve this documentation. If you have any problems or unclarities, feel free to open an issue.
