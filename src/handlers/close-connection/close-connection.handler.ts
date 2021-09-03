import { getCliSpace } from '../../shell-utils';
import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';
import { ConnectionService } from '../../services/connection/connection.service';
import { ConnectionState } from '../../services/connection/connection.types';
import { SessionService } from '../../services/session/session.service';

export async function closeConnectionHandler(
    configService: ConfigService,
    logger: Logger,
    connectionId: string,
    closeAll: boolean
){
    const sessionService = new SessionService(configService, logger);
    const cliSpace = await getCliSpace(sessionService, logger);
    if(! cliSpace){
        logger.error(`There is no cli session. Try creating a new connection to a target using the zli`);
        await cleanExit(1, logger);
    }
    const connectionService = new ConnectionService(configService, logger);

    if(closeAll)
    {
        logger.info('Closing all connections open in cli-space');
        await sessionService.CloseSession(cliSpace.id);
        await sessionService.CreateSession('cli-space');
    } else {
        const conn = await connectionService.GetConnection(connectionId);
        // if the connection does belong to the cli space
        if (conn.sessionId !== cliSpace.id){
            logger.error(`Connection ${connectionId} does not belong to the cli space`);
            await cleanExit(1, logger);
        }
        // if connection not already closed
        if(conn.state == ConnectionState.Open){
            await connectionService.CloseConnection(connectionId);
            logger.info(`Connection ${connectionId} successfully closed`);
        }else{
            logger.error(`Connection ${connectionId} is not open`);
            await cleanExit(1, logger);
        }
    }

    await cleanExit(0, logger);
}