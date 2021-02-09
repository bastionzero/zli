# Helper script to get token and session id
# Ensure you've set your dev ServiceURL and run:
# `zli --configName "dev" login Google`
import subprocess
import re
import json
import sys


def getToken(zliPath, config_name):
    # If we don't pass a custom zli path, use the normal `zli`
    if (zliPath is None):
        zliPath = 'zli'

    # Run our zli command to get the configPath
    # get the first line only since we also print the log file config path as well
    configInfo = str(subprocess.check_output(f"{zliPath} config --configName {config_name}", shell=True).strip().splitlines()[0])
    pattern = r"You can edit your config here: ((?:[^/]*/)*(.*)[^\"])"
    configPath = re.search(pattern, configInfo).group(1)[:-1] # there is some ' at the end, drop it

    # Load in the information from the configPath
    try:
        with open(configPath) as configFile:
            configData = json.load(configFile)
    except Exception as e:
        print(f"Error loading config file: {configPath}")
        raise e

    # Print the id_token and sessionId
    try:
        print(f"ID TOKEN:\n{configData['tokenSet']['id_token']}")
        print(f"SESSION ID:\n{configData['sessionId']}")
    except Exception as e: 
        print("Error parsing config file, did you run `zli --configName dev login ...` yet?")
        raise e
