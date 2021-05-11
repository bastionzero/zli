import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { SessionService, ConnectionService, PolicyQueryService } from '../http.service/http.service';
import { ConnectionState, VerbType } from '../http.service/http.service.types';
import { ParsedTargetString, SessionState, TargetType } from '../types';
import { ShellTerminal } from '../terminal/terminal';
import { MixpanelService } from '../mixpanel.service/mixpanel.service';
import { cleanExit } from './clean-exit.handler';

import termsize from 'term-size';
import { targetStringExampleNoPath } from '../utils';
import _ from 'lodash';


export async function connectHandler(
    configService: ConfigService,
    logger: Logger,
    mixpanelService: MixpanelService,
    parsedTarget: ParsedTargetString) {

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

    // call list session
    const sessionService = new SessionService(configService, logger);
    const listSessions = await sessionService.ListSessions();

    // space names are not unique, make sure to find the latest active one
    const cliSpace = listSessions.sessions.filter(s => s.displayName === 'cli-space' && s.state == SessionState.Active); // TODO: cli-space name can be changed in config

    // maybe make a session
    let cliSessionId: string;
    if(cliSpace.length === 0) {
        cliSessionId =  await sessionService.CreateSession('cli-space');
    } else {
        // there should only be 1 active 'cli-space' session
        cliSessionId = cliSpace.pop().id;
    }

    // We do the following for ssh since we are required to pass
    // in a user although it does not get read at any point
    const targetUser = parsedTarget.type === TargetType.SSH ? 'ssh' : parsedTarget.user;

    // make a new connection
    const connectionService = new ConnectionService(configService, logger);
    // if SSM user does not exist then resp.connectionId will throw a
    // 'TypeError: Cannot read property 'connectionId' of undefined'
    // so we need to catch and return undefined
    const connectionId = await connectionService.CreateConnection(parsedTarget.type, parsedTarget.id, cliSessionId, targetUser).catch(() => undefined);

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

    // connect to target and run terminal
    const terminal = new ShellTerminal(logger, configService, connectionService, connectionId);
    try {
        await terminal.start(termsize());
    } catch (err) {
        logger.error(`Error connecting to terminal: ${err.stack}`);
        await cleanExit(1, logger);
    }

    mixpanelService.TrackNewConnection(parsedTarget.type);

    // Terminal resize event logic
    // https://nodejs.org/api/process.html#process_signal_events -> SIGWINCH
    // https://github.com/nodejs/node/issues/16194
    // https://nodejs.org/api/process.html#process_a_note_on_process_i_o
    process.stdout.on(
        'resize',
        () => {
            const resizeEvent = termsize();
            terminal.resize(resizeEvent);
        }
    );

    terminal.terminalRunning.subscribe(
        () => {},
        // If an error occurs in the terminal running observable then log the
        // error, clean up the connection, and exit zli
        async (error) => {
            logger.error(error);
            terminal.dispose();

            logger.debug('Cleaning up connection...');
            const conn = await connectionService.GetConnection(connectionId);
            // if connection not already closed
            if(conn.state == ConnectionState.Open)
                await connectionService.CloseConnection(connectionId);

            logger.debug('Connection closed');

            await cleanExit(1, logger);
        },
        // If terminal running observable completes without error, exit zli
        // without closing the connection
        async () => {
            terminal.dispose();
            await cleanExit(0, logger);
        }
    );

    // To get 'keypress' events you need the following lines
    // ref: https://nodejs.org/api/readline.html#readline_readline_emitkeypressevents_stream_interface
    const readline = require('readline');
    readline.emitKeypressEvents(process.stdin);
    if (process.stdin.isTTY) {
        process.stdin.setRawMode(true);
    }
    process.stdin.on('keypress', (_, key) => terminal.writeString(key.sequence));
}