#!/bin/bash

# Start the process
echo "Starting Snap2..."
start-stop-daemon --start \
  --no-close --background \
  --make-pidfile --pidfile /tmp/snaps.pid \
  --exec /usr/local/bin/snap2

# Keep checking health
sleep 10

while true; do
  /usr/local/bin/healthcheck.sh &>/dev/null
  if [ $? -ne 0 ] ; then
    echo "HEALTHCHECK FAILED. Exiting..."
    echo $?
    exit 1
  fi
  sleep 5
done
