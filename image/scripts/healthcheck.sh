#!/bin/bash

# HEALTHCHECK FOR DOCKER USE
# EXIT STATUS:
#   0: all proccesses are running
#   1: one of the proccesses is not running

for cmd in peer snapsd; do
  PID=$(pidof ${cmd})
  echo -n "Checking ${cmd}... "
  if [ -z "${PID}" ]; then
    echo "FAIL"
    exit 1
  fi
  echo "OK"
done

exit 0
