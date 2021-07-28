import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
const { spawn } = require('child_process');



export async function disconnectHandler(
    configService: ConfigService,
    logger: Logger
) {
    // First get the pid from the config service
    var kubeConfig = configService.getKubeConfig();

    // then kill the daemon
    if (kubeConfig['localPid'] != null) {
        // First try to kill the process
        spawn('pkill', ['-P', kubeConfig['localPid'].toString()])

        // Update the config
        kubeConfig['localPid'] = null
        configService.setKubeConfig(kubeConfig);
        
        logger.info('Killed local Kube daemon')
    } else {
        logger.warn('No Kube daemon running')
    }
}