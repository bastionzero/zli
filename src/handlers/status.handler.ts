import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import { cleanExit } from './clean-exit.handler';

export async function statusHandler(
    configService: ConfigService,
    logger: Logger
) {
    // First get the status from the config service
    var kubeConfig = configService.getKubeConfig();

    // then kill the daemon
    if (kubeConfig['localPid'] == null) {
        logger.warn('No Kube daemon running')
    } else {
        // Pull the info from the config and show it to the user
        logger.info(`Kube Daemon running:`)
        logger.info(`    - Assume Cluster: ${kubeConfig['assumeCluster']}`)
        logger.info(`    - Assume Role: ${kubeConfig['assumeRole']}`)
        logger.info(`    - Local URL: ${kubeConfig['localHost']}:${kubeConfig['localPort']}`)
    }
    await cleanExit(0, logger)
}