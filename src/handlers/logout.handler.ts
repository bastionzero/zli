import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';
const { spawn } = require('child_process');


export async function logoutHandler(configService: ConfigService, logger: Logger) {
    // Deletes the auth tokens from the config which will force the
    // user to login again before running another command
    configService.logout();
    logger.info('Closing any existing SSH Tunnel Connections');

    logger.info('Closing any existing Kube Proxy Connections');
    var kubeConfig = configService.getKubeConfig();
    if (kubeConfig['localPid'] != null) {
        // First try to kill the process
        spawn('pkill', ['-P', kubeConfig['localPid'].toString()])

        // Update the config
        kubeConfig['localPid'] = null
        configService.setKubeConfig(kubeConfig);
    }

    logger.info('Logout successful');
    await cleanExit(0, logger);
}