import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { getTableOfConnections } from '../../utils';
import { cleanExit } from '../clean-exit.handler';
import { getCliSpace } from '../../shell-utils';
import { TargetSummary } from '../../services/common.types';
import { ConnectionState, ConnectionDetails } from '../../services/connection/connection.types';
import { SessionService } from '../../services/session/session.service';
import yargs from 'yargs';
import { listConnectionsArgs } from './list-connections.command-builder';

export async function listConnectionsHandler(
    argv: yargs.Arguments<listConnectionsArgs>,
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