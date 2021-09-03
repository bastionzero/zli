import { Logger } from '../../services/logger/logger.service';
import { ConfigService } from '../../services/config/config.service';
import { cleanExit } from '../clean-exit.handler';
import { killDaemon } from '../../services/kube/kube.service';



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