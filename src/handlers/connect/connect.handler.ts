import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';

import { targetStringExample } from '../../utils';
import { createAndRunShell, getCliSpace } from '../../shell-utils';
import { includes } from 'lodash';
import { ParsedTargetString } from '../../services/common.types';
import { ConnectionService } from '../../services/connection/connection.service';
import { MixpanelService } from '../../services/mixpanel/mixpanel.service';
import { PolicyQueryService } from '../../services/policy-query/policy-query.service';
import { VerbType } from '../../services/policy-query/policy-query.types';
import { SessionService } from '../../services/session/session.service';


export async function connectHandler(
    configService: ConfigService,
    logger: Logger,
    mixpanelService: MixpanelService,
    parsedTarget: ParsedTargetString
) {

    if(! parsedTarget) {
        logger.error('No targets matched your targetName/targetId or invalid target string, must follow syntax:');
        logger.error(targetStringExample);
        await cleanExit(1, logger);
    }

    const policyQueryService = new PolicyQueryService(configService, logger);
    const response = await policyQueryService.ListTargetOSUsers(parsedTarget.id, parsedTarget.type, {type: VerbType.Shell}, undefined);

    if(! response.allowed)
    {
        logger.error('You do not have sufficient permission to access the target');
        await cleanExit(1, logger);
    }

    const allowedTargetUsers = response.allowedTargetUsers.map(u => u.userName);
    if(response.allowedTargetUsers && ! includes(allowedTargetUsers, parsedTarget.user)) {
        logger.error(`You do not have permission to connect as targetUser: ${parsedTarget.user}`);
        logger.info(`Current allowed users for you: ${allowedTargetUsers}`);
        await cleanExit(1, logger);
    }

    // Get the existing if any or create a new cli space id
    const sessionService = new SessionService(configService, logger);
    const cliSpace = await getCliSpace(sessionService, logger);
    let cliSpaceId: string;
    if (cliSpace === undefined)
    {
        cliSpaceId = await sessionService.CreateSession('cli-space');
    } else {
        cliSpaceId = cliSpace.id;
    }

    const targetUser = parsedTarget.user;

    // make a new connection
    const connectionService = new ConnectionService(configService, logger);
    // if SSM user does not exist then resp.connectionId will throw a
    // 'TypeError: Cannot read property 'connectionId' of undefined'
    // so we need to catch and return undefined
    const connectionId = await connectionService.CreateConnection(parsedTarget.type, parsedTarget.id, cliSpaceId, targetUser).catch(() => undefined);

    if(! connectionId)
    {
        logger.error('Connection creation failed');

        logger.error(`You may not have a policy for targetUser ${parsedTarget.user} in environment ${parsedTarget.envName}`);
        logger.info('You can find SSM user policies in the web app');

        await cleanExit(1, logger);
    }

    // Note: For DATs the actual target to connect to will be a dynamically
    // created ssm target that is provisioned by the DynamicAccessTarget and not
    // the id of the dynamic access target. The dynamically created ssm target should be
    // returned in the connectionSummary.targetId for this newly created
    // connection

    const connectionSummary = await connectionService.GetConnection(connectionId);

    await createAndRunShell(configService, logger, connectionSummary);

    mixpanelService.TrackNewConnection(parsedTarget.type);
}