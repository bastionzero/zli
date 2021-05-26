import { ConnectionService, SessionService } from '../../src/http.service/http.service';
import { ConfigService } from '../../src/config.service/config.service';
import { Logger } from '../../src/logger.service/logger';
import { ConnectionState } from '../../src/http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';
import { createAndRunShell, getCliSpaceId } from '../../src/shell-utils';

export async function attachHandler(
    configService: ConfigService,
    logger: Logger,
    connectionId: string
){
    const connectionService = new ConnectionService(configService, logger);
    const connectionSummary = await connectionService.GetConnection(connectionId);

    const sessionService = new SessionService(configService, logger);
    const cliSessionId = await getCliSpaceId(sessionService, logger);

    if ( ! cliSessionId){
        logger.error(`There is no cli session. Try creating a new connection to a target using the zli`);
        await cleanExit(1, logger);
    }
    if (connectionSummary.sessionId !== cliSessionId){
        logger.error(`Connection ${connectionId} does not belong to the cli space`);
        await cleanExit(1, logger);
    }
    if (connectionSummary.state !== ConnectionState.Open){
        logger.error(`Connection ${connectionId} is not open`);
        await cleanExit(1, logger);
    }
    await createAndRunShell(configService, logger, connectionSummary.id, connectionService);
}