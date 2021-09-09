import path from 'path';
import utils from 'util';
import fs from 'fs';
import { killDaemon } from '../../services/kube/kube.service';
import { ClusterDetails, KubeClusterStatus } from '../../services/kube/kube.types';
import { PolicyQueryService } from '../../services/policy-query/policy-query.service';
import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';
import { LoggerConfigService } from '../../services/logger/logger-config.service';
import yargs from 'yargs';
import { tunnelArgs } from './tunnel.command-builder';
const { spawn } = require('child_process');

const WINDOWS_DAEMON_PATH : string = '../../../../bctl-go-daemon/bctl/daemon/daemon-windows';
const LINUX_DAEMON_PATH : string = '../../../../bctl-go-daemon/bctl/daemon/daemon-linux';
const MACOS_DAEMON_PATH : string = '../../../../bctl-go-daemon/bctl/daemon/daemon-macos';

export async function startKubeDaemonHandler(argv: yargs.Arguments<tunnelArgs>, assumeUser: string, assumeCluster: string, clusterTargets: Promise<ClusterDetails[]>, configService: ConfigService, logger: Logger, loggerConfigService: LoggerConfigService) {
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

    // Make sure the user has created a kubeConfig before
    if (kubeConfig == undefined) {
        logger.error('Please make sure you have created your kubeconfig before running proxy. You can do this via "zli generate kubeConfig"');
        await cleanExit(1, logger);
    }


    if (kubeConfig['localPid'] != null) {
        killDaemon(configService);
    }

    // See if the user passed in a custom port
    let daemonPort = kubeConfig['localPort'].toString();
    if (argv.customPort != -1) {
        daemonPort = argv.customPort.toString();
    }

    // Build our args and cwd
    let args = [
        `-sessionId=${configService.sessionId()}`,
        `-assumeRole=${assumeUser}`,
        `-assumeClusterId=${clusterTarget.id}`,
        `-daemonPort=${daemonPort}`,
        `-serviceURL=${configService.serviceUrl().slice(0, -1).replace('https://', '')}`,
        `-authHeader="${configService.getAuthHeader()}"`,
        `-localhostToken="${kubeConfig['token']}"`,
        `-environmentId="${clusterTarget.environmentId}"`,
        `-certPath="${kubeConfig['certPath']}"`,
        `-keyPath="${kubeConfig['keyPath']}"`,
        `-configPath=${configService.configPath()}`,
        `-logPath="${loggerConfigService.daemonLogPath()}"`
    ];
    let cwd = process.cwd();


    // Copy over our executable to a temp file
    let finalDaemonPath = '';
    if (process.env.ZLI_CUSTOM_BCTL_PATH) {
        // If we set a custom path, we will try to start the daemon from the source code
        cwd = process.env.ZLI_CUSTOM_BCTL_PATH;
        finalDaemonPath = 'go';
        args = ['run', 'daemon.go'].concat(args);
    } else {
        finalDaemonPath = await copyExecutableToLocalDir(logger, configService.configPath());
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
                if (process.platform === 'linux') {
                    spawn('pkill', ['-s', daemonProcess.pid], {
                        cwd: process.cwd(),
                        shell: true,
                        detached: true,
                        stdio: 'inherit'
                    });
                } else {
                    spawn('kill', ['-9', daemonProcess.pid], {
                        cwd: process.cwd(),
                        shell: true,
                        detached: true,
                        stdio: 'inherit'
                    });
                }
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

async function getClusterInfoFromName(clusterTargets: ClusterDetails[], clusterName: string, logger: Logger): Promise<ClusterDetails> {
    for (const clusterTarget of clusterTargets) {
        if (clusterTarget.name == clusterName) {
            return clusterTarget;
        }
    }
    logger.error('Unable to find cluster!');
    await cleanExit(1, logger);
}

async function copyExecutableToLocalDir(logger: Logger, configPath: string): Promise<string> {
    // Helper function to copy the Daemon executable to a local dir on the file system
    // Ref: https://github.com/vercel/pkg/issues/342

    // First get the parent dir of the config path
    const configFileDir = path.dirname(configPath);

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

    let daemonExecPath = undefined;
    let finalDaemonPath = undefined;
    if (process.platform === 'win32') {
        daemonExecPath = path.join(__dirname, WINDOWS_DAEMON_PATH);

        finalDaemonPath = path.join(configFileDir, 'daemon-windows.exe');
    }
    else if (process.platform === 'linux' || process.platform === 'darwin') {
        if (process.platform === 'linux') {
            daemonExecPath = path.join(__dirname, LINUX_DAEMON_PATH);
        } else {
            daemonExecPath = path.join(__dirname, MACOS_DAEMON_PATH);
        }

        finalDaemonPath = path.join(configFileDir, 'daemon');
    } else {
        logger.error(`Unsupported operating system: ${process.platform}`);
        await cleanExit(1, logger);
    }

    await deleteIfExists(finalDaemonPath);

    // Create our executable file
    fs.writeFileSync(finalDaemonPath, '');

    // Copy the file to the computers file system
    await copy(daemonExecPath, finalDaemonPath);

    // Grant execute permission
    await chmod(finalDaemonPath, 0o755);

    // Return the path
    return finalDaemonPath;
}


async function deleteIfExists(pathToFile: string) {
    // Check if the file exists, delete if so
    if (fs.existsSync(pathToFile)) {
        // Delete the file
        fs.unlinkSync(pathToFile);
    }
}