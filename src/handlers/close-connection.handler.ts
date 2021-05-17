import { getCliSpaceId } from '../../src/shell-utils';
import { ConfigService } from '../../src/config.service/config.service';
import { ConnectionService, SessionService } from '../../src/http.service/http.service';
import { ConnectionState } from '../../src/http.service/http.service.types';
import { Logger } from '../../src/logger.service/logger';
import { cleanExit } from './clean-exit.handler';

export async function closeConnectionHandler(
    configService: ConfigService,
    logger: Logger,
    connectionId: string
){
    const sessionService = new SessionService(configService, logger);
    const cliSessionId = await getCliSpaceId(sessionService, logger);
    if ( ! cliSessionId){
        logger.error(`There is no cli session. Try creating a new connection to a target using the zli`);
        await cleanExit(1, logger);
    }
    const connectionService = new ConnectionService(configService, logger);
    logger.debug('Cleaning up connection...');
    const conn = await connectionService.GetConnection(connectionId);
    // if the connection does belong to the cli space
    if (conn.sessionId !== cliSessionId){
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
    await cleanExit(0, logger);
}