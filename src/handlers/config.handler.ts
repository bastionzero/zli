import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { LoggerConfigService } from '../logger-config.service/logger-config.service';
import { cleanExit } from './clean-exit.handler';


export async function configHandler(logger: Logger, configService: ConfigService, loggerConfigService: LoggerConfigService) {
    logger.info(`You can edit your config here: ${configService.configPath()}`);
    logger.info(`You can find your log files here: ${loggerConfigService.logPath()}`);
    await cleanExit(0, logger);
}