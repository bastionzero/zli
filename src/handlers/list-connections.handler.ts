import { ConfigService } from '../../src/config.service/config.service';
import { Logger } from '../../src/logger.service/logger';
import { SessionService } from '../../src/http.service/http.service';
import { SessionState, TargetSummary } from '../types';
import { getTableOfConnections } from '../../src/utils';
import { cleanExit } from './clean-exit.handler';
import { ConnectionState } from '../../src/http.service/http.service.types';

export async function listConnectionsHandler(
    configService: ConfigService,
    logger: Logger,
    ssmTargets: Promise<TargetSummary[]>,
    sshTargets: Promise<TargetSummary[]>
){
    // await and concatenate
    const allTargets = [...await ssmTargets, ...await sshTargets];
    // call list session
    const sessionService = new SessionService(configService, logger);
    const listSessions = await sessionService.ListSessions();

    // space names are not unique, make sure to find the latest active one
    const cliSpace = listSessions.sessions.filter(s => s.displayName === 'cli-space' && s.state == SessionState.Active); // TODO: cli-space name can be changed in config

    let cliSessionId: string;
    if(cliSpace.length === 0) {
        logger.info('There is no cli session available');
        await cleanExit(0, logger);
    } else {
        // there should only be 1 active 'cli-space' session
        cliSessionId = cliSpace.pop().id;
    }

    const sessionDetails = await sessionService.GetSession(cliSessionId);
    const openConnections = sessionDetails.connections.filter(c => c.state === ConnectionState.Open);
    if (openConnections.length === 0){
        logger.info('There are no open zli connections');
        await cleanExit(0, logger);
    }
    const tableString = getTableOfConnections(openConnections, allTargets);
    console.log(tableString);

    await cleanExit(0, logger);
}