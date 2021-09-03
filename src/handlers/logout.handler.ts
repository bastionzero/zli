import { killDaemon } from '../services/kube/kube.service';
import { ConfigService } from '../services/config/config.service';
import { Logger } from '../services/logger/logger.service';
import { cleanExit } from './clean-exit.handler';


export async function logoutHandler(configService: ConfigService, logger: Logger) {
    // Deletes the auth tokens from the config which will force the
    // user to login again before running another command
    configService.logout();
    logger.info('Closing any existing SSH Tunnel Connections');

    logger.info('Closing any existing Kube Proxy Connections');
    const kubeConfig = configService.getKubeConfig();
    if (kubeConfig !== null && kubeConfig['localPid'] !== null) {
        killDaemon(configService);
    }

    logger.info('Logout successful');
    await cleanExit(0, logger);
}