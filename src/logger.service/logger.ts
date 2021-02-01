import chalk from 'chalk';
import figlet from 'figlet';
import winston, { Logger as WinstonLogger, format } from 'winston';
import { LoggerConfigService } from '../logger-config.service/logger-config.service';
const { printf } = format;

// Not an enum, must be dictionary for winston
const thoumLoggingLevels = {
    Error: 0,
    Warn: 1,
    Info: 2,
    Debug: 3,
    Trace: 4
};
const thoumLoggingColors: { [level: string]: string; } = {
    Error: "\x1b[31m",
    Warn: "\x1b[33m",
    Info: "\x1b[35m",
    Debug: "\x1b[36m",
    Trace: "\x1b[37m"
};
const thoumLoggingDefaultFormat = printf(info => {
    return `${thoumLoggingColors[info.level]}${info.message}\x1b[0m`;
});
const thoumLoggingDebugFormat = printf(info => {
    return `${thoumLoggingColors[info.level]}[${[info.level]}][${info.timestamp}] ${info.message}\x1b[0m`;
});


export class Logger {
    private debugFlag: boolean;
    private silentFlag: boolean;
    private logger: WinstonLogger;
    private config: LoggerConfigService;

    constructor(config: LoggerConfigService, debug: boolean, silent: boolean) {
        this.debugFlag = debug;
        this.config = config;
        this.silentFlag = silent;

        // Build our logger
        this.buildLogger();
    }

    private buildLogger(): void {
        // Helper function to build our logger
        try {
            this.logger = winston.createLogger({
                levels: thoumLoggingLevels,
                format: winston.format.combine(
                    winston.format.timestamp({
                        format: 'YYYY-MM-DD HH:mm:ss'
                    }),
                    winston.format.errors({ stack: true }),
                    winston.format.splat(),
                    winston.format.json()
                ),
                transports: [
                    new winston.transports.File({
                        level: "Trace",
                        filename: this.config.logPath(),
                    })
                ]
            });
        } catch (error) {
            if (error.code == 'EACCES') {
                // This would happen if the user does not have access to create dirs in /var/log/
                var errorMessage = `Please ensure that you have access to ${this.config.logPath()}`;
            } else {
                // Else it's an unknown error
                var errorMessage = `${error.message}`;
            }
            console.log(chalk.red(`${errorMessage}`));
            process.exit(1);
        }

        if (!this.silentFlag) {
            // If the silent flag has not been passed

            if (this.debugFlag) {
                // If we're debugging, ensure that all levels are being streamed to the console
                this.logger.add(new winston.transports.Console({
                    level: "Trace",
                    format: thoumLoggingDebugFormat
                }));
                // Display our title if we are debugging
                this.displayTitle();
            } else {
                // If we are not debugging, we want info and above to transport to the console with our custom format
                this.logger.add(new winston.transports.Console({
                    level: "Info",
                    format: thoumLoggingDefaultFormat
                }));
            }
        }

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
                figlet.textSync('clunk80 cli', { horizontalLayout: 'full' })
            )
        );
        this.info(`You can find all logs here: ${this.config.logPath()}`);
    }
}