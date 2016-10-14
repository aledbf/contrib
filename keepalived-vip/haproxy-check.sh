#!/bin/sh

# this script checks the status of HAProxy verifying the http status code from the HAProxy dashboard 
CODE=$(curl --write-out %{http_code} --silent --output /dev/null http://127.0.0.1:1936)
if [ $CODE -eq 200 ]; then
  exit 0
else
  exit 1
fi