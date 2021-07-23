import sys
import subprocess

import uuid

def main():
    # Extract the args
    argsForKube = f'{" ".join(sys.argv[1:])}'

    # Append the args to the token in the kubeconfig using the zli
    token = subprocess.check_output('/Users/sidpremkumar/Documents/CommonwealthCrypto/zli/bin/zli-macos --configName dev get-kube-token -s', shell=True).decode('utf-8').strip()

    # Also generate and add logId so we can match all Api calls with this command
    logId = str(uuid.uuid4()) 
    formattedToken = f'{token}bctl {argsForKube}++++{logId}'

    # Make the kubectl call with the token being the args so we can extract in Bastion
    subprocess.Popen(f'kubectl --token "{formattedToken}" {argsForKube}', shell=True).communicate()