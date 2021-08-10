import path from 'path';
import { killDaemon } from '../../src/kube.service/kube.service';
import { ConfigService } from '../config.service/config.service';
import { PolicyQueryService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { ClusterSummary, KubeClusterStatus } from '../types';
import { cleanExit } from './clean-exit.handler';
const { spawn } = require('child_process');
const fs = require('fs');
const utils = require('util');
const tmp = require('tmp');

export async function startKubeDaemonHandler(argv: any, assumeUser: string, assumeCluster: string, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
    // First check that the cluster is online
    const clusterTarget = await getClusterInfoFromName(await clusterTargets, assumeCluster, logger);
    if (clusterTarget.status != KubeClusterStatus.Online) {
        logger.error('Target cluster is offline!');
        await cleanExit(1, logger);
    }

    // Make our API client
    const policyService = new PolicyQueryService(configService, logger);

    // Now check that the user has the correct OPA permissions (we will do this again when the daemon starts)
    const response = await policyService.CheckKubeProxy(assumeCluster, assumeUser, clusterTarget.environmentId);
    if (response.allowed != true) {
        logger.error(`You do not have the correct policy setup to access ${assumeCluster} as ${assumeUser}`);
        await cleanExit(1, logger);
    }

    // Check if we've already started a process
    const kubeConfig = configService.getKubeConfig();
    // TODO : Make sure the user has created a kubeConfig before
    if (kubeConfig == undefined) {
        logger.error('Please make sure you have created your kubeconfig before running proxy. You can do this via "zli generate kubeConfig"');
        await cleanExit(1, logger);
    }


    if (kubeConfig['localPid'] != null) {
        killDaemon(configService);
    }

    // Build our args and cwd
    let args = [`-sessionId=${configService.sessionId()}`, `-assumeRole=${assumeUser}`, `-assumeClusterId=${clusterTarget.id}`, `-daemonPort=${kubeConfig['localPort']}`, `-serviceURL=${configService.serviceUrl().slice(0, -1).replace('https://', '')}`, `-authHeader="${configService.getAuthHeader()}"`, `-localhostToken="${kubeConfig['token']}"`, `-environmentId="${clusterTarget.environmentId}"`, `-certPath="${kubeConfig['certPath']}"`, `-keyPath="${kubeConfig['keyPath']}"`];
    let cwd = process.cwd();


    // Copy over our executable to a temp file
    let finalDaemonPath = '';
    if (process.env.ZLI_CUSTOM_BCTL_PATH) {
        // If we set a custom path, we will try to start the daemon from the source code
        cwd = process.env.ZLI_CUSTOM_BCTL_PATH;
        finalDaemonPath = 'go';
        args = ['run', 'main.go'].concat(args);
    } else {
        finalDaemonPath = await copyExecutableToTempDir();
    }

    try {
        if (!argv.debug) {
            // If we are not debugging, start the go subprocess in the background
            const options = {
                cwd: cwd,
                detached: true,
                shell: true,
                stdio: ['ignore', 'ignore', 'ignore']
            };

            const daemonProcess = await spawn(finalDaemonPath, args, options);

            // Now save the Pid so we can kill the process next time we start it
            kubeConfig['localPid'] = daemonProcess.pid;

            // Save the info about assume cluster and role
            kubeConfig['assumeRole'] = assumeUser;
            kubeConfig['assumeCluster'] = assumeCluster;
            configService.setKubeConfig(kubeConfig);

            logger.info(`Started kube daemon at ${kubeConfig['localHost']}:${kubeConfig['localPort']} for ${assumeUser}@${assumeCluster}`);
            process.exit(0);
        } else {
            // Start our daemon process, but stream our stdio to the user (pipe)
            const daemonProcess = await spawn(finalDaemonPath, args,
                {
                    cwd: cwd,
                    shell: true,
                    detached: true,
                    stdio: 'inherit'
                }
            );

            process.on('SIGINT', () => {
                // CNT+C Sent from the user, kill the daemon process, which will trigger an exit
                spawn('pkill', ['-P', daemonProcess.pid], {
                    cwd: process.cwd(),
                    shell: true,
                    detached: true,
                    stdio: 'inherit'
                });
            });

            daemonProcess.on('exit', function () {
                // Whenever the daemon exits, exit
                process.exit();
            });
        }
    } catch (error) {
        logger.error(`Something went wrong starting the Kube Daemon: ${error}`);
        await cleanExit(1, logger);
    }
}

async function getClusterInfoFromName(clusterTargets: ClusterSummary[], clusterName: string, logger: Logger): Promise<ClusterSummary> {
    for (const clusterTarget of clusterTargets) {
        if (clusterTarget.name == clusterName) {
            return clusterTarget;
        }
    }
    logger.error('Unable to find cluster!');
    await cleanExit(1, logger);
}

async function copyExecutableToTempDir(): Promise<string> {
    // Helper function to copy the Daemon executable to a temp dir on the file system
    // Ref: https://github.com/vercel/pkg/issues/342
    const chmod = utils.promisify(fs.chmod);

    // Our copy function as we cannot use fs.copyFileSync
    async function copy(source: string, target: string) {
        return new Promise<void>(async function (resolve, reject) {
            const ret = await fs.createReadStream(source).pipe(fs.createWriteStream(target), { end: true });
            ret.on('close', () => {
                resolve();
            });
            ret.on('error', () => {
                reject();
            });
        });

    }

    // We have to go up 1 more directory bc when we compile we are inside /dist
    const daemonExecPath = path.join(__dirname, '../../../bctl-go-daemon/Daemon/Daemon');

    // Create our temp file
    const tmpobj = tmp.fileSync();
    const finalDaemonPath = `${tmpobj.name}`;

    // Copy the file to the computers file system
    await copy(daemonExecPath, finalDaemonPath); // this should work

    // Grant execute permission
    // TODO: See if this is the right level of permission
    await chmod(finalDaemonPath, 0o765);

    // Return the path
    return finalDaemonPath;
}