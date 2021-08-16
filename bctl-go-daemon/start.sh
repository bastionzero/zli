#!/bin/sh
if [ $DEV == "true" ]; then
    echo "Dev set to $DEV...sleeping forever"
    sleep infinity
else
    /bctl-agent/agent -serviceUrl=$SERVICE_URL
fi