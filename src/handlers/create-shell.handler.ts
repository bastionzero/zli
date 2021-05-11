import { ConfigService } from '../../src/config.service/config.service';
import { Logger } from '../../src/logger.service/logger';
import { ShellTerminal } from '../../src/terminal/terminal';
import { TargetType } from '../types';
import { cleanExit } from './clean-exit.handler';
import termsize from 'term-size';

export async function createShellHandler(
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