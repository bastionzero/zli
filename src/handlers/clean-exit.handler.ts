import { Logger } from '../services/logger/logger.service';


export async function cleanExit(exitCode: number, logger: Logger) {
    await logger.flushLogs();
    process.exit(exitCode);
}