import { ConfigService } from '../../src/config.service/config.service';
import { ConnectionService } from '../../src/http.service/http.service';
import { ConnectionState } from '../../src/http.service/http.service.types';
import { Logger } from '../../src/logger.service/logger';
import { cleanExit } from './clean-exit.handler';

export async function closeConnectionHandler(
    configService: ConfigService,
    logger: Logger,
    connectionId: string
){
    const connectionService = new ConnectionService(configService, logger);
    logger.debug('Cleaning up connection...');
    const conn = await connectionService.GetConnection(connectionId);
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