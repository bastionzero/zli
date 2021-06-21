import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { ConnectionService, PolicyQueryService, SessionService } from '../http.service/http.service';
import { VerbType } from '../http.service/http.service.types';
import { ParsedTargetString, TargetType } from '../types';
import { MixpanelService } from '../mixpanel.service/mixpanel.service';
import { cleanExit } from './clean-exit.handler';

import { targetStringExampleNoPath } from '../utils';
import { createAndRunShell, getCliSpace } from '../../src/shell-utils';
import _ from 'lodash';


export async function connectHandler(
    configService: ConfigService,
    logger: Logger,
    mixpanelService: MixpanelService,
    parsedTarget: ParsedTargetString
) {

    if(! parsedTarget) {
        logger.error('No targets matched your targetName/targetId or invalid target string, must follow syntax:');
        logger.error(targetStringExampleNoPath);
        await cleanExit(1, logger);
    }

    const policyQueryService = new PolicyQueryService(configService, logger);
    const response = await policyQueryService.ListTargetUsers(parsedTarget.id, parsedTarget.type, {type: VerbType.Shell}, undefined);

    if(! response.allowed)
    {
        logger.error('You do not have sufficient permission to access the target');
        await cleanExit(1, logger);
    }

    const allowedTargetUsers = response.allowedTargetUsers.map(u => u.userName);
    if(response.allowedTargetUsers && ! _.includes(allowedTargetUsers, parsedTarget.user)) {
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

    // We do the following for ssh since we are required to pass
    // in a user although it does not get read at any point
    const targetUser = parsedTarget.type === TargetType.SSH ? 'ssh' : parsedTarget.user;

    // make a new connection
    const connectionService = new ConnectionService(configService, logger);
    // if SSM user does not exist then resp.connectionId will throw a
    // 'TypeError: Cannot read property 'connectionId' of undefined'
    // so we need to catch and return undefined
    const connectionId = await connectionService.CreateConnection(parsedTarget.type, parsedTarget.id, cliSpaceId, targetUser).catch(() => undefined);

    if(! connectionId)
    {
        logger.error('Connection creation failed');
        if(parsedTarget.type !== TargetType.SSH)
        {
            logger.error(`You may not have a policy for targetUser ${parsedTarget.user} in environment ${parsedTarget.envName}`);
            logger.info('You can find SSM user policies in the web app');
        } else {
            logger.info('Please check your polices in the web app for this target and/or environment');
        }

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