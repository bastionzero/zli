import {
    parseTargetString,
    targetStringExample
} from '../utils';
import { FileService } from '../http.service/http.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';

import fs from 'fs';


export async function copyHandler(configService: ConfigService, logger: Logger, argv: any) {
    const fileService = new FileService(configService, logger);

    const sourceParsedString = parseTargetString(argv.targetType, argv.source);
    const destParsedString = parseTargetString(argv.targetType, argv.destination);
    const parsedTarget = sourceParsedString || destParsedString; // one of these will be undefined so javascript will use the other

    // figure out upload or download
    // would be undefined if not parsed properly
    if(destParsedString)
    {
        // Upload case
        // First ensure that the file exists
        if (!fs.existsSync(argv.source)) {
            logger.warn(`File ${argv.source} does not exist!`);
            process.exit(1);
        }

        // Then create our read stream and try to upload it
        const fh = fs.createReadStream(argv.source);
        await fileService.uploadFile(parsedTarget.id, parsedTarget.type, parsedTarget.path, fh, parsedTarget.user);
        logger.info('File upload complete');

    } else if(sourceParsedString) {
    // download case
        await fileService.downloadFile(parsedTarget.id, parsedTarget.type, parsedTarget.path, argv.destination, parsedTarget.user);
    } else {
        logger.error('Invalid target string, must follow syntax:');
        logger.error(targetStringExample);
    }
    await cleanExit(0, logger);
}