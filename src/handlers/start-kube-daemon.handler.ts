import {
    HubConnection,
    HubConnectionBuilder,
    HubConnectionState,
    LogLevel,
} from "@microsoft/signalr";
import { ConfigService } from '../config.service/config.service';
import { SignalRLogger } from "../../webshell-common-ts/logging/signalr-logger";
import { Logger } from "../logger.service/logger";
import { v4 as uuidv4 } from 'uuid';

const { spawn } = require('child_process');
const kill = require('tree-kill');

// const https = require('https');
// const fs = require('fs');
// const WebSocket = require('ws');
// const spdy = require('spdy')
// const spdyp2p = require('libp2p-spdy')

// const pull = require('pull-stream')
// const toPull = require('stream-to-pull-stream')
// const tcp = require('net')


export async function startKubeDaemonHandler(connectUser: string, connectCluster: string, configService: ConfigService, logger: Logger) {
    // First check if we've already started a process
    var kubeConfig = configService.getKubeConfig();
    if (kubeConfig['localPid'] != null) {
        // TODO: Add check here to ensure the process exists
        // First try to kill the process
        // TODO: process.kill doesnt seem to kill the child process', not sure why :(
        spawn('pkill', ['-P', kubeConfig['localPid'].toString()])
    }

    // Start the go subprocess
    // TODO: This will chance when we compile it inside zli
    const options = {
        cwd: "/Users/sidpremkumar/Documents/CommonwealthCrypto/Random/bctl-go-daemon/",
        // detached: true,
        // shell: false,
        stdio: ['ignore', 'ignore', 'ignore']
    };

    // Build our args 
    let args = ['run', 'main.go', `-sessionId=${configService.sessionId()}`, `-assumeRole=${connectUser}`, `-assumeCluster=${connectCluster}`, `-daemonPort=${kubeConfig['localPort']}`, `-serviceURL=${configService.serviceUrl().slice(0, -1).replace("https://", "")}`, `-authHeader="${configService.getAuthHeader()}"`]
    console.log(`go ${args.join(' ')}`)

    // Testing websocket
    // const socket = await buildWebsocket(configService, connectUser, logger)
    // socket.start();
    // const delay = (ms: any) => new Promise(resolve => setTimeout(resolve, ms))
    // await delay(2000) /// waiting 1 second.



    // const daemonProcess = spawn('go', ['run', 'main.go'], options);

    // // Now save the Pid so we can kill the process next time we start it
    // kubeConfig["localPid"] = daemonProcess.pid;
    // configService.setKubeConfig(kubeConfig);
    // console.log(daemonProcess)

    // logger.info(`Started kube daemon at ${kubeConfig["localHost"]}:${kubeConfig['localPort']} for ${connectUser}@${connectCluster}`);

    // console.log(daemonProcess.pid);
    // process.exit(0)
}

async function generateUUID(): Promise<string> {
    return uuidv4();
}

async function generateUniqueId(): Promise<number> {
    // Helper function to generate a uniqueId
    // ret number: unique number
    return Math.round(Math.random() * 1000);
}

async function buildWebsocket(configService: ConfigService, connectUser: string, logger: Logger): Promise<HubConnection> {
    const connectionBuilder = new HubConnectionBuilder();
    connectionBuilder.withUrl(
    `${configService.serviceUrl()}api/v1/hub/kube?session_id=${configService.sessionId()}&assume_role=${connectUser}`,
    {
        accessTokenFactory: () => configService.getAuth(),
    }
    )
    .configureLogging(new SignalRLogger(logger))
    .withAutomaticReconnect()
    .configureLogging(LogLevel.None);
    return connectionBuilder.build();
}