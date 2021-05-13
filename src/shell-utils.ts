import termsize from 'term-size';
import { ConfigService } from './config.service/config.service';
import { cleanExit } from './handlers/clean-exit.handler';
import { SessionService } from './http.service/http.service';
import { Logger } from './logger.service/logger';
import { ShellTerminal } from './terminal/terminal';
import { SessionState, TargetType } from './types';

export async function createAndRunShell(
    configService: ConfigService,
    logger: Logger,
    targetType: TargetType,
    targetId: string,
    connectionId: string
){
    // connect to target and run terminal
    const terminal = new ShellTerminal(logger, configService, connectionId, targetType, targetId);
    try {
        await terminal.start(termsize());
    } catch (err) {
        logger.error(`Error connecting to terminal: ${err.stack}`);
        await cleanExit(1, logger);
    }

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

export async function getCliSpaceId(
    sessionService: SessionService,
    logger: Logger
): Promise<string> {
    const listSessions = await sessionService.ListSessions();

    // space names are not unique, make sure to find the latest active one
    const cliSpace = listSessions.sessions.filter(s => s.displayName === 'cli-space' && s.state == SessionState.Active); // TODO: cli-space name can be changed in config

    let cliSessionId: string;
    if(cliSpace.length === 0) {
        return undefined;
    } else if (cliSpace.length === 1) {
        cliSessionId = cliSpace[0].id;
    } else {
        // there should only be 1 active 'cli-space' session
        cliSessionId = cliSpace.pop().id;
        logger.warn(`Found ${cliSpace.length} cli sessions while expecting 1`);
    }
    return cliSessionId;
}