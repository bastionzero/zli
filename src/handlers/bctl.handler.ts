import { ConfigService } from '../../src/config.service/config.service';
import { Logger } from '../../src/logger.service/logger';
import { cleanExit } from './clean-exit.handler';
import util from 'util';
import { spawn, exec } from 'child_process';

const { v4: uuidv4 } = require('uuid');
const execPromise = util.promisify(exec);


export async function bctlHandler(configService: ConfigService, logger: Logger, listOfCommands: string[]) {
    // Check if daemon is even running
    const kubeConfig = configService.getKubeConfig();
    if (kubeConfig['localPid'] == null) {
        logger.warn('No Kube daemon running');
        await cleanExit(1, logger);
    }

    // Print as what user we are running the command as, and to which container
    logger.info(`Connected as ${kubeConfig['assumeRole']} to cluster ${kubeConfig['assumeCluster']}`);

    // Then get the token
    const token = kubeConfig['token'];

    // Now generate a log id
    const logId = uuidv4();

    // Now build our token
    const kubeArgsString = listOfCommands.join(' ');

    // We use '++++' as a delimiter so that we can parse the engligh command, logId, token in the daemon
    const formattedToken = `${token}++++zli kube ${kubeArgsString}++++${logId}`;

    // Add the token to the args
    let kubeArgs: string[] = ['--token', formattedToken];

    // Then add the extract the args
    kubeArgs = kubeArgs.concat(listOfCommands);

    const kubeCommandProcess = await spawn('kubectl', kubeArgs, { stdio: [process.stdin, process.stdout, process.stderr] });

    kubeCommandProcess.on('close', async (code: number) => {
        logger.debug(`Kube command process exited with code ${code}`);

        if (code != 0) {
            // Check to ensure they are using the right context
            const currentContext = await execPromise('kubectl config current-context ');

            if (currentContext.stdout != 'bctl-server') {
                logger.warn('Make sure you using the correct kube config!');
            }
        }
    });
}