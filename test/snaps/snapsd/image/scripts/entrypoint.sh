#!/bin/bash

# Start the first process
echo "Starting Fabric Peer..."
start-stop-daemon --start \
  --no-close --background \
  --make-pidfile --pidfile /tmp/peer.pid \
  --exec /usr/local/bin/peer -- node start --peer-defaultchain=false

# Start the second process
echo "Starting Snaps..."
start-stop-daemon --start \
  --no-close --background \
  --make-pidfile --pidfile /tmp/snaps.pid \
  --exec /usr/local/bin/snapsd

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
