import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';
import { createAndRunShell, getCliSpace } from '../../shell-utils';
import { ConnectionService } from '../../services/connection/connection.service';
import { ConnectionState } from '../../services/connection/connection.types';
import { SessionService } from '../../services/session/session.service';

export async function attachHandler(
    configService: ConfigService,
    logger: Logger,
    connectionId: string
){
    const connectionService = new ConnectionService(configService, logger);
    const connectionSummary = await connectionService.GetConnection(connectionId);

    const sessionService = new SessionService(configService, logger);
    const cliSpace = await getCliSpace(sessionService, logger);

    if ( ! cliSpace){
        logger.error(`There is no cli session. Try creating a new connection to a target using the zli`);
        await cleanExit(1, logger);
    }
    if (connectionSummary.sessionId !== cliSpace.id){
        logger.error(`Connection ${connectionId} does not belong to the cli space`);
        await cleanExit(1, logger);
    }
    if (connectionSummary.state !== ConnectionState.Open){
        logger.error(`Connection ${connectionId} is not open`);
        await cleanExit(1, logger);
    }
    return await createAndRunShell(configService, logger, connectionSummary);
}