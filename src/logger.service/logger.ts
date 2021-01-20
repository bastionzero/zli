import { thoumError, thoumMessage, thoumWarn } from './../utils';
import chalk from 'chalk';
import figlet from 'figlet';
import winston, { Logger as WinstonLogger } from 'winston';
import { LoggerConfigService } from '../logger-config.service/logger-config.service';

// Not an enum, must be dictionary for winston
const thoumLoggingLevels = {
    error:  4,
    warn:   3,
    info:   2,
    debug:  1,
    trace:  0
};

export class Logger {
    private debugFlag: boolean;
    private logger: WinstonLogger;
    private config: LoggerConfigService;

    constructor(config: LoggerConfigService, debug: boolean){
        this.debugFlag = debug;
        this.config = config

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
                        filename: this.config.logPath(),
                      })
                ]
              });
        } catch (error) {
            if (error.code == 'EACCES') {
                // This would happen if the user does not have access to create dirs in /var/log/
                thoumError(`Please ensure that you have access to ${this.config.logPath()}`)
            } else {
                // Else it's an unknown error
                thoumError(`${error.message}`)
            }
            process.exit(1);
        }

        if (this.debugFlag) {
            // Display our title
            this.displayTitle();

            // If we're debugging, ensure that all levels are being streamed to the console
            this.logger.add(new winston.transports.Console({
              format: winston.format.simple()
            }));
        }
    }

    public info(message: string): void {
        this.logger.info(message);

        // If we're not debugging, we want to pretty print it
        if (!this.debugFlag) {
            thoumMessage(message);
        }
    }

    public debug(message: string): void {
        this.logger.debug(message);
    }

    public error(message: string): void {
        this.logger.error(message);
        
        // If we're not debugging,  we want to pretty print it
        if (!this.debugFlag) {
            thoumError(message);
        }
    }

    public warn(message: string): void {
        this.logger.warn(message);
        
        // If we're not debugging,  we want to pretty print it
        if (!this.debugFlag) {
            thoumWarn(message);
        }
    }

    public trace(message: string): void {
        this.logger.log('trace', message);
    }

    public displayTitle(): void {
        console.log(
            chalk.magentaBright(
                figlet.textSync('clunk80 cli', { horizontalLayout: 'full' })
            )
        );
        thoumMessage(`You can find all logs here: ${this.config.logPath()}`)
    }

}