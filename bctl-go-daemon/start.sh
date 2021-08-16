#!/bin/sh
if [ $DEV == "true" ]; then
    sleep infinity
else
    ./agent -serviceURL=$SERVICE_URL
fi