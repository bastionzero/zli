import { ConfigService } from '../../config.service/config.service';
import { Logger } from '../../logger.service/logger';
import { SessionService } from '../../http.service/http.service';
import { ConnectionDetails, TargetSummary } from '../../types';
import { getTableOfConnections } from '../../utils';
import { cleanExit } from '../clean-exit.handler';
import { ConnectionState } from '../../http.service/http.service.types';
import { getCliSpace } from '../../shell-utils';

export async function listConnectionsHandler(
    argv: any,
    configService: ConfigService,
    logger: Logger,
    ssmTargets: Promise<TargetSummary[]>,
){
    const sessionService = new SessionService(configService, logger);
    const cliSpace = await getCliSpace(sessionService, logger);

    const openConnections = cliSpace.connections.filter(c => c.state === ConnectionState.Open);

    // await and concatenate
    const allTargets = [...await ssmTargets];
    const formattedConnections = openConnections.map<ConnectionDetails>((conn, _index, _array) => {
        return {
            id: conn.id,
            timeCreated: conn.timeCreated,
            targetId: conn.serverId,
            sessionId: conn.sessionId,
            state: conn.state,
            serverType: conn.serverType,
            userName: conn.userName
        };
    });

    if(!! argv.json) {
        // json output
        console.log(JSON.stringify(formattedConnections));
    } else {
        if (formattedConnections.length === 0){
            logger.info('There are no open zli connections');
            await cleanExit(0, logger);
        }
        // regular table output
        const tableString = getTableOfConnections(formattedConnections, allTargets);
        console.log(tableString);
    }

    await cleanExit(0, logger);
}