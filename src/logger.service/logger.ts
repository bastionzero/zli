import { thoumError, thoumMessage, thoumWarn } from './../utils';
import chalk from 'chalk';
import figlet from 'figlet';
import winston, { Logger as WinstonLogger } from 'winston';
import path from 'path';

const thoumLoggingLevel = {
    error:  3,
    warn:   2,
    info:   1,
    debug:  0
};

const LOG_PATH = '/var/log/thoum.log'

export class Logger {
    private debugFlag: boolean;
    private logger: WinstonLogger;

    constructor(debug: boolean){
        // Set our debug flag
        this.debugFlag = debug;

        // Build our logger
        this.logger = winston.createLogger({
            levels: thoumLoggingLevel,
            format: winston.format.combine(
                winston.format.timestamp({
                format: 'YYYY-MM-DD HH:mm:ss'
              }),
              winston.format.errors({ stack: true }),
              winston.format.splat(),
              winston.format.json()
            ),
            defaultMeta: { service: 'thoum' },
            transports: [
                new winston.transports.File({
                    filename: path.join(LOG_PATH),
                  })
            ]
          });


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

    public displayTitle(): void {
        console.log(
            chalk.magentaBright(
                figlet.textSync('clunk80 cli', { horizontalLayout: 'full' })
            )
        );
        thoumMessage(`You can find all logs here: ${LOG_PATH}`)
    }

}