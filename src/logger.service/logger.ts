import chalk from 'chalk';
import figlet from 'figlet';
import winston, { Logger as WinstonLogger, format } from 'winston';
import { LoggerConfigService } from '../logger-config.service/logger-config.service';
import { ILogger } from '../../webshell-common-ts/logging/logging.types';
const { printf } = format;

// Not an enum, must be dictionary for winston
const loggingLevels = {
    Error: 0,
    Warn: 1,
    Info: 2,
    Debug: 3,
    Trace: 4
};
const loggingColors: { [level: string]: string; } = {
    Error: '\x1b[31m',
    Warn: '\x1b[33m',
    Info: '\x1b[35m',
    Debug: '\x1b[36m',
    Trace: '\x1b[37m'
};
const loggingDefaultFormat = printf(info => {
    return `${loggingColors[info.level]}${info.message}\x1b[0m`;
});
const loggingDebugFormat = printf(info => {
    return `${loggingColors[info.level]}[${[info.level]}][${info.timestamp}] ${info.message}\x1b[0m`;
});


export class Logger implements ILogger {
    private debugFlag: boolean;
    private silentFlag: boolean;
    private logger: WinstonLogger;
    private config: LoggerConfigService;

    constructor(config: LoggerConfigService, debug: boolean, silent: boolean, isStdEnabled: boolean) {
        this.debugFlag = debug;
        this.config = config;
        this.silentFlag = silent;

        // Build our logger
        this.buildLogger(isStdEnabled);
    }

    private buildLogger(isStdEnabled: boolean): void {
        // Helper function to build our logger
        const transportsOptions = {
            file: new winston.transports.File({
                level: 'Debug',
                filename: this.config.logPath()
            }),
            stderr: new winston.transports.Stream({
                stream: process.stderr,
                level: 'Error',
                format: loggingDefaultFormat
            })
        };
        let transports = undefined;
        // If we do not have control of the stdio we have to connect manually
        // the log.error() to the stderr in order to print error messages
        if (!isStdEnabled){
            transports = [transportsOptions.file, transportsOptions.stderr];
        }else{
            transports = [transportsOptions.file];
        }
        try {
            this.logger = winston.createLogger({
                levels: loggingLevels,
                format: winston.format.combine(
                    winston.format.timestamp({
                        format: 'YYYY-MM-DD HH:mm:ss'
                    }),
                    winston.format.errors({ stack: true }),
                    winston.format.splat(),
                    winston.format.json()
                ),
                transports: transports,
            });
        } catch (error) {
            let errorMessage;
            if (error.code == 'EACCES') {
                // This would happen if the user does not have access to create dirs in /var/log/
                errorMessage = `Please ensure that you have access to ${this.config.logPath()}`;
            } else {
                // Else it's an unknown error
                errorMessage = `${error.message}`;
            }
            console.log(chalk.red(`${errorMessage}`));
            process.exit(1);
        }

        if (!this.silentFlag) {
            // If the silent flag has not been passed

            if (this.debugFlag) {
                // If we're debugging, ensure that all levels are being streamed to the console
                this.logger.add(new winston.transports.Console({
                    level: 'Trace',
                    format: loggingDebugFormat
                }));
                // Display our title if we are debugging
                this.displayTitle();
            } else {
                // If we are not debugging, we want info and above to transport to the console with our custom format
                this.logger.add(new winston.transports.Console({
                    level: 'Info',
                    format: loggingDefaultFormat
                }));
            }
        }

        this.logger.on('error', (_) => {
            // suppress errors that occur in the logger otherwise they will be
            // unhandled exceptions this may happen if we write to the logger
            // after having called flushLogs which ends the log stream
        });
    }

    // Ends the logger stream and waits for finish event to be emitted
    public flushLogs(): Promise<void> {
        return new Promise((resolve, _) => {
            this.logger.on('finish', () => {
                // This should work without a timeout but this is currently an open bug in winston
                // https://github.com/winstonjs/winston#awaiting-logs-to-be-written-in-winston
                // https://github.com/winstonjs/winston/issues/1504
                setTimeout(() => resolve(), 1000);
            });
            this.logger.end();
        });
    }

    public info(message: string): void {
        this.logger.log('Info', message);
    }

    public error(message: string): void {
        this.logger.log('Error', message);
    }

    public warn(message: string): void {
        this.logger.log('Warn', message);
    }

    public debug(message: string): void {
        this.logger.log('Debug', message);
    }

    public trace(message: string): void {
        this.logger.log('Trace', message);
    }

    public displayTitle(): void {
        console.log(
            chalk.magentaBright(
                figlet.textSync('bzero', { horizontalLayout: 'full' })
            )
        );
        this.info(`You can find all logs here: ${this.config.logPath()}`);
    }
}