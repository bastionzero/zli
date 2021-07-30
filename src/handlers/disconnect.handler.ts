import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import { cleanExit } from './clean-exit.handler';
import { killDaemon } from '../../src/kube.service/kube.service';
const { spawn } = require('child_process');



export async function disconnectHandler(
    configService: ConfigService,
    logger: Logger
) {
    // First get the pid from the config service
    var kubeConfig = configService.getKubeConfig();

    // then kill the daemon
    if (kubeConfig['localPid'] != null) {
        killDaemon(configService);
        
        logger.info('Killed local Kube daemon')
    } else {
        logger.warn('No Kube daemon running')
    }

    await cleanExit(0, logger);
}