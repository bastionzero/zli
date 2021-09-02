import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import { cleanExit } from './clean-exit.handler';
import { killDaemon } from '../../src/kube.service/kube.service';



export async function disconnectHandler(
    configService: ConfigService,
    logger: Logger
) {
    if (await killDaemon(configService)) {
        logger.info('Killed local Kube daemon');
    } else {
        logger.warn('No Kube daemon running');
    }
    await cleanExit(0, logger);
}