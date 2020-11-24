#!/bin/bash

Version=$(node -p "require('./package.json').version")

echo "{
    \"version\": \"$Version\"
}"