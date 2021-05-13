import { ConfigService } from '../../src/config.service/config.service';
import { Logger } from '../../src/logger.service/logger';
import { SessionService } from '../../src/http.service/http.service';
import { ConnectionDetails, SessionState, TargetSummary } from '../types';
import { getTableOfConnections } from '../../src/utils';
import { cleanExit } from './clean-exit.handler';
import { ConnectionState } from '../../src/http.service/http.service.types';

export async function listConnectionsHandler(
    argv: any,
    configService: ConfigService,
    logger: Logger,
    ssmTargets: Promise<TargetSummary[]>,
    sshTargets: Promise<TargetSummary[]>,
    cliSpaceId: Promise<string>
){
    const cliSessionId = await cliSpaceId;
    const sessionService = new SessionService(configService, logger);
    const sessionDetails = await sessionService.GetSession(cliSessionId);
    const openConnections = sessionDetails.connections.filter(c => c.state === ConnectionState.Open);
    if (openConnections.length === 0){
        logger.info('There are no open zli connections');
        await cleanExit(0, logger);
    }
    // await and concatenate
    const allTargets = [...await ssmTargets, ...await sshTargets];
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
        // regular table output
        const tableString = getTableOfConnections(formattedConnections, allTargets);
        console.log(tableString);
    }

    await cleanExit(0, logger);
}