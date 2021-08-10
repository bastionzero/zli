import { ConsoleLogger } from "@microsoft/signalr/dist/esm/Utils";
import { ConfigService } from "../../src/config.service/config.service";
import { Logger } from "../../src/logger.service/logger";
import { cleanExit } from "./clean-exit.handler";

const { v4: uuidv4 } = require('uuid');
const spawn = require('child_process').spawn;


export async function bctlHandler(configService: ConfigService, logger: Logger) {
    // Check if daemon is even running
    var kubeConfig = configService.getKubeConfig();
    if (kubeConfig["localPid"] == null) {
        logger.warn('No Kube daemon running');
        await cleanExit(1, logger);
    }

    // Print as what user we are running the command as, and to which container
    logger.info(`Connected as ${kubeConfig["assumeRole"]} to cluster ${kubeConfig["assumeCluster"]}`)

    // Then get the token 
    const token = kubeConfig['token'];

    // Now generate a log id
    const logId = uuidv4();

    // Now build our token
    const kubeArgsRaw = process.argv.splice(2)
    const kubeArgsString = kubeArgsRaw.join(' ');
    const formattedToken = `${token}bctl ${kubeArgsString}++++${logId}`;

    // Add the token to the args
    var kubeArgs: string[] = ['--token', formattedToken]

    // Then add the extract the args
    kubeArgs = kubeArgs.concat(kubeArgsRaw);

    spawn('kubectl', kubeArgs, { stdio: [process.stdin, process.stdout, process.stderr] });
}