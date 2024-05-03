#!/bin/bash

if [[ -z "${WORKER_MODE}" ]]; then
  if [[ "${WORKER_MODE}" == "true" ]]; then
    # Run as worker
    /app/run-worker.sh
  else 
    # Run as manager
    /app/main
  fi
else
    # Run as manager
    /app/main
fi