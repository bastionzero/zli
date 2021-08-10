import { ConsoleLogger } from "@microsoft/signalr/dist/esm/Utils";
import { ConfigService } from "../../src/config.service/config.service";
import { Logger } from "../../src/logger.service/logger";

const { v4: uuidv4 } = require('uuid');
const spawn = require('child_process').spawn;


export async function bctlHandler(configService: ConfigService, logger: Logger) {
    // First extract the args
    var kubeArgs = process.argv.splice(2);

    // Then get the token 
    var kubeConfig = configService.getKubeConfig();

    const token = kubeConfig['token'];

    // Now generate a log id
    const logId = uuidv4();
    console.log(kubeArgs);

    // Now build our token
    const kubeArgsString = kubeArgs.join(' ');
    const formattedToken = `${token}bctl ${kubeArgsString}++++${logId}`;

    // Add the token to the args
    kubeArgs.unshift('--token');
    kubeArgs.upshift(formattedToken);






    // spawn('kubectl', kubeArgs, { stdio: [process.stdin, process.stdout, process.stderr] });
}





// const getTokenProcess = spawnSync('zli', ['--configName', 'dev', 'get-kube-token', '-s']);
// const token = getTokenProcess.stdout.toString('utf8');




