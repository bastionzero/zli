import { Logger } from '../logger.service/logger';


export async function cleanExit(exitCode: number, logger: Logger) {
    await logger.flushLogs();
    process.exit(exitCode);
}