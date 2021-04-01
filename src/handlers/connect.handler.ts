import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { SessionService, ConnectionService } from '../http.service/http.service';
import { EnvironmentDetails, ConnectionState } from '../http.service/http.service.types';
import { SessionState, TargetType } from '../types';
import {
    TargetSummary,
    parsedTargetString,
    parseTargetString,
    checkTargetTypeAndStringPair,
    getTableOfTargets,
    targetStringExampleNoPath
} from '../utils';
import { ShellTerminal } from '../terminal/terminal';
import { MixpanelService } from '../mixpanel.service/mixpanel.service';
import { cleanExit } from './clean-exit.handler';

import termsize from 'term-size';


export async function connectHandler(
    configService: ConfigService,
    logger: Logger,
    argv: any,
    mixpanelService: MixpanelService,
    dynamicConfigs: Promise<TargetSummary[]>,
    ssmTargets: Promise<TargetSummary[]>,
    sshTargets: Promise<TargetSummary[]>,
    envs: Promise<EnvironmentDetails[]>) {
    const parsedTarget = await disambiguateTargetName(argv.targetType, argv.targetString, logger, dynamicConfigs, ssmTargets, sshTargets, envs);

    // call list session
    const sessionService = new SessionService(configService, logger);
    const listSessions = await sessionService.ListSessions();

    // space names are not unique, make sure to find the latest active one
    const cliSpace = listSessions.sessions.filter(s => s.displayName === 'cli-space' && s.state == SessionState.Active); // TODO: cli-space name can be changed in config

    // maybe make a session
    let cliSessionId: string;
    if(cliSpace.length === 0)
    {
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
            const allSsmBasedTargets = (await ssmTargets).concat(await dynamicConfigs);
            const targetEnvId = allSsmBasedTargets.filter(ssm => ssm.id == parsedTarget.id)[0].environmentId;
            const targetEnvName = (await envs).filter(e => e.id == targetEnvId)[0].name;
            logger.error(`You may not have a policy for targetUser ${parsedTarget.user} in environment ${targetEnvName}`);
            logger.info('You can find SSM user policies in the web app');
        } else {
            logger.info('Please check your polices in the web app for this target and/or environment');
        }

        await cleanExit(1, logger);
    }

    // connect to target and run terminal
    const terminal = new ShellTerminal(logger, configService, connectionId);
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

    // If we detect a disconnection, close the connection immediately
    terminal.terminalRunning.subscribe(
        () => {},
        async (error) => {
            if(error)
            {
                logger.error(error);
                logger.warn('Target may have gone offline or space/connection closed from another client');
            }

            terminal.dispose();

            logger.debug('Cleaning up connection...');
            const conn = await connectionService.GetConnection(connectionId);
            // if connection not already closed
            if(conn.state == ConnectionState.Open)
                await connectionService.CloseConnection(connectionId);

            logger.debug('Connection closed');

            if(error)
                await cleanExit(1, logger);

            await cleanExit(0, logger);
        },
        () => {}
    );

    // To get 'keypress' events you need the following lines
    // ref: https://nodejs.org/api/readline.html#readline_readline_emitkeypressevents_stream_interface
    const readline = require('readline');
    readline.emitKeypressEvents(process.stdin);
    process.stdin.setRawMode(true);
    process.stdin.on('keypress', (_, key) => terminal.writeString(key.sequence));
}


// Figure out target id based on target name and target type.
// Also preforms error checking on target type and target string passed in
export async function disambiguateTargetName(
    argvTargetType: string,
    argvTargetString: string,
    logger: Logger,
    dynamicConfigs: Promise<TargetSummary[]>,
    ssmTargets: Promise<TargetSummary[]>,
    sshTargets: Promise<TargetSummary[]>,
    envs: Promise<EnvironmentDetails[]>): Promise<parsedTargetString> {
    const parsedTarget = parseTargetString(argvTargetType, argvTargetString);

    if (!parsedTarget) {
        logger.error('Invalid target string, must follow syntax:');
        logger.error(targetStringExampleNoPath);
        await cleanExit(1, logger);
    }

    if (!checkTargetTypeAndStringPair(parsedTarget)) {

        switch (parsedTarget.type) {
        case TargetType.SSH:
            logger.warn('Cannot specify targetUser for SSH connections');
            logger.warn('Please try your previous command without the targetUser');
            logger.warn('Target string for SSH: targetId[:path]');
            break;
        case TargetType.SSM:
        case TargetType.DYNAMIC:
            logger.warn('Must specify targetUser for SSM and DYNAMIC connections');
            logger.warn('Target string for SSM: targetUser@targetId[:path]');
            break;
        default:
            throw new Error('Unhandled TargetType');
        }
        await cleanExit(1, logger);
    }


    if (parsedTarget.name) {
        let matchedNamedTargets: TargetSummary[] = [];

        switch (parsedTarget.type) {
        case TargetType.SSM:
            matchedNamedTargets = (await ssmTargets).filter(ssm => ssm.name === parsedTarget.name);
            break;
        case TargetType.SSH:
            matchedNamedTargets = (await sshTargets).filter(ssh => ssh.name === parsedTarget.name);
            break;
        case TargetType.DYNAMIC:
            matchedNamedTargets = (await dynamicConfigs).filter(config => config.name === parsedTarget.name);
            break;
        default:
            logger.error(`Invalid TargetType passed ${parsedTarget.type}`);
            await cleanExit(1, logger);
        }

        if (matchedNamedTargets.length < 1) {
            logger.error(`No ${parsedTarget.type} targets found with name ${parsedTarget.name}`);
            logger.warn('Target names are case sensitive');
            logger.warn('To see list of all targets run: \'zli lt\'');
            await cleanExit(1, logger);
        } else if (matchedNamedTargets.length == 1) {
            // the rest of the flow will work as before since the targetId has now been disambiguated
            parsedTarget.id = matchedNamedTargets.pop().id;
        } else {
            // ambiguous target id, warn user, exit process
            logger.warn(`Multiple ${parsedTarget.type} targets found with name ${parsedTarget.name}`);
            const tableString = getTableOfTargets(matchedNamedTargets, await envs);
            console.log(tableString);
            logger.info('Please connect using \'target id\' instead of target name');
            await cleanExit(1, logger);
        }
    }

    return parsedTarget;
}