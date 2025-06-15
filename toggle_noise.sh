#!/bin/zsh

# Check if the brown_noise process is running
pid=$(pgrep -f "./brown_noise")

if [ -z "$pid" ]; then
  # If the process is not running, start it
  nohup ./brown_noise > /dev/null 2>&1 &
  echo "Brown noise started."
else
  # If the process is running, kill it
  kill "$pid"
  echo "Brown noise stopped."
fi
