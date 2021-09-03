import { Observable } from 'rxjs';
import { bufferTime, filter } from 'rxjs/operators';
import termsize from 'term-size';
import { ConfigService } from './services/config/config.service';
import { cleanExit } from './handlers/clean-exit.handler';
import { Logger } from './services/logger/logger.service';
import { ShellTerminal } from './terminal/terminal';
import { ConnectionSummary } from './services/connection/connection.types';
import { SessionService } from './services/session/session.service';
import { SessionDetails, SessionState } from './services/session/session.types';

export async function createAndRunShell(
    configService: ConfigService,
    logger: Logger,
    connectionSummary: ConnectionSummary
){
    // connect to target and run terminal
    const terminal = new ShellTerminal(logger, configService, connectionSummary);
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

    const source = new Observable<string>(function (observer) {
        // To get 'keypress' events you need the following lines
        // ref: https://nodejs.org/api/readline.html#readline_readline_emitkeypressevents_stream_interface
        const readline = require('readline');
        readline.emitKeypressEvents(process.stdin);
        if (process.stdin.isTTY) {
            process.stdin.setRawMode(true);
        }
        process.stdin.on('keypress', (_, key) => observer.next(key.sequence));
    });
    source.pipe(bufferTime(50), filter(buffer => buffer.length > 0)).subscribe(keypresses => {
        // This pipe is in order to allow copy-pasted input to be treated like a single string
        terminal.writeString(keypresses.join(''));
    });
}

export async function getCliSpace(
    sessionService: SessionService,
    logger: Logger
): Promise<SessionDetails> {
    const listSessions = await sessionService.ListSessions();

    // space names are not unique, make sure to find the latest active one
    const cliSpace = listSessions.sessions.filter(s => s.displayName === 'cli-space' && s.state == SessionState.Active); // TODO: cli-space name can be changed in config

    if(cliSpace.length === 0) {
        return undefined;
    } else if (cliSpace.length === 1) {
        return cliSpace[0];
    } else {
        // there should only be 1 active 'cli-space' session
        logger.warn(`Found ${cliSpace.length} cli spaces while expecting 1, using latest one`);
        return cliSpace.pop();
    }
}