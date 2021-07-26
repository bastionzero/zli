import {
    HubConnection,
    HubConnectionBuilder,
    HubConnectionState,
    LogLevel,
} from "@microsoft/signalr";
import { ConfigService } from '../config.service/config.service';
import { SignalRLogger } from "../../webshell-common-ts/logging/signalr-logger";
import { PolicyQueryService } from '../http.service/http.service';
import { Logger } from "../logger.service/logger";
import { v4 as uuidv4 } from 'uuid';
import { ClusterSummary, KubeClusterStatus } from "../types";
import { cleanExit } from './clean-exit.handler';
const { spawn } = require('child_process');


export async function startKubeDaemonHandler(connectUser: string, connectCluster: string, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
    // First check that the cluster is online 
    var clusterTarget = await getClusterInfoFromName(await clusterTargets, connectCluster, logger);
    if (clusterTarget.status != KubeClusterStatus.Online) {
        logger.error('Target cluster is offline!');
        await cleanExit(1, logger);
    }

    // Make our API client
    const policyService = new PolicyQueryService(configService, logger);

    // Now check that the user has the correct OPA permissions (we will do this again when the daemon starts)
    var response = await policyService.CheckKubeProxy(connectCluster, connectUser, clusterTarget.environmentId);
    if (response.allowed != true) {
        logger.error(`You do not have the correct policy setup to access ${connectCluster} as ${connectUser}`);
        await cleanExit(1, logger);
    }

    // Check if we've already started a process
    var kubeConfig = configService.getKubeConfig();
    // TODO : Make sure the user has created a kubeConfig before
    if (kubeConfig['localPid'] != null) {
        // First try to kill the process
        spawn('pkill', ['-P', kubeConfig['localPid'].toString()])
    }

    // Start the go subprocess
    // TODO: This will chance when we compile it inside zli
    const options = {
        cwd: "/Users/sidpremkumar/Documents/CommonwealthCrypto/zli/bctl-go-daemon/Daemon",
        detached: true,
        shell: true,
        stdio: ['ignore', 'ignore', 'ignore']
    };

    // Build our args 
    let args = ['run', 'main.go', `-sessionId=${configService.sessionId()}`, `-assumeRole=${connectUser}`, `-assumeCluster=${connectCluster}`, `-daemonPort=${kubeConfig['localPort']}`, `-serviceURL=${configService.serviceUrl().slice(0, -1).replace("https://", "")}`, `-authHeader="${configService.getAuthHeader()}"`, `-localhostToken="${kubeConfig['token']}"`, `-environmentId="${clusterTarget.environmentId}"`]

    console.log(`go ${args.join(' ')}`)
    // const daemonProcess = await spawn('go', args, options);

    // // Now save the Pid so we can kill the process next time we start it
    // kubeConfig["localPid"] = daemonProcess.pid;
    // configService.setKubeConfig(kubeConfig);

    // logger.info(`Started kube daemon at ${kubeConfig["localHost"]}:${kubeConfig['localPort']} for ${connectUser}@${connectCluster}`);
    // process.exit(0)
}

async function getClusterInfoFromName(clusterTargets: ClusterSummary[], clusterName: string, logger: Logger): Promise<ClusterSummary> {
    for (var clusterTarget of clusterTargets) {
        if (clusterTarget.name == clusterName) {
            return clusterTarget;
        }
    }
    logger.error('Unable to find cluster!');
    await cleanExit(1, logger);
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