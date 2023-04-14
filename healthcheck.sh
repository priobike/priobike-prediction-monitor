#!/bin/bash

# Get the status json
json_data=`cat ${STATIC_PATH}status.json`

# Check if the reading of the file succeeded
if [ $? -ne 0 ]; then
    # Unhealthy
    echo "Error getting status.json."
    exit 1
fi

# Extract status_update_time "status_update_time" timestamp from the JSON data
status_update_time=$(echo "$json_data" | jq -r '.status_update_time')

# Check if the extraction succeeded
if [ $? -ne 0 ]; then
    # Unhealthy
    echo "Error extracting timestamp from JSON data."
    exit 1
fi

# Get the current timestamp in seconds since epoch
current_seconds=$(date +%s)

# Calculate the time difference in seconds
time_difference=$((current_seconds - status_update_time))

# If the time difference is greater than 5 minutes (300 seconds), exit with code 1
if [ $time_difference -gt 300 ]; then
    # Unhealthy
    echo "The status update timestamp is older than 5 minutes."
    exit 1
else
    # Extract most_recent_prediction_time "most_recent_prediction_time" timestamp from the JSON data
    first_most_recent_prediction_time=$(echo "$json_data" | jq -r '.most_recent_prediction_time')

    # Check if the extraction succeeded
    if [ $? -ne 0 ]; then
        # Unhealthy
        echo "Error extracting timestamp from JSON data."
        exit 1
    fi

    # Wait 80 seconds.
    sleep 80

    # Get the status json
    json_data_two=`cat ${STATIC_PATH}status.json`

    # Check if the reading of the file succeeded
    if [ $? -ne 0 ]; then
        # Unhealthy
        echo "Error getting status.json."
        exit 1
    fi

    # Extract most_recent_prediction_time "most_recent_prediction_time" timestamp from the JSON data
    second_most_recent_prediction_time=$(echo "$json_data_two" | jq -r '.most_recent_prediction_time')

    # Check if the extraction succeeded
    if [ $? -ne 0 ]; then
        # Unhealthy
        echo "Error extracting timestamp from JSON data."
        exit 1
    fi

    if (( first_most_recent_prediction_time == second_most_recent_prediction_time )); then
        # Unhealthy
        echo "The most_recent_prediction_time doesn't get updated."
        exit 1
    else
        echo "The most_recent_prediction_time get's updated normally."
        exit 0
    fi
fi
