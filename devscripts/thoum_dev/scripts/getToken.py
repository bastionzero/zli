# Helper script to get token and session id
# Ensure you've set your dev ServiceURL and run:
# `thoum --configName "dev" login Google`
import subprocess
import re
import json
import sys


def getToken(thoumPath, config_name):
    # If we don't pass a custom thoum path, use the normal `thoum`
    if (thoumPath is None):
        thoumPath = 'thoum'

    # Run our thoum command to get the configPath
    # get the first line only since we also print the log file config path as well
    configInfo = str(subprocess.check_output(f"{thoumPath} config --configName {config_name}", shell=True).strip().splitlines()[0])
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
        print("Error parsing config file, did you run `thoum --configName \"dev\" login ...` yet?")
        raise e
