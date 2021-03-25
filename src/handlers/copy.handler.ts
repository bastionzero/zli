import {
    parseTargetString,
    targetStringExample,
    TargetSummary
} from '../utils';
import { FileService } from '../http.service/http.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';

import fs from 'fs';
import { disambiguateTargetName } from './connect.handler';
import { EnvironmentDetails } from '../http.service/http.service.types';
import { TargetType } from '../types';


export async function copyHandler(
    configService: ConfigService,
    logger: Logger,
    argv: any,
    dynamicConfigs: Promise<TargetSummary[]>,
    ssmTargets: Promise<TargetSummary[]>,
    sshTargets: Promise<TargetSummary[]>,
    envs: Promise<EnvironmentDetails[]>) {

    const sourceParsedString = parseTargetString(argv.targetType, argv.source);
    const destParsedString = parseTargetString(argv.targetType, argv.destination);

    if(! sourceParsedString && ! destParsedString)
    {
        logger.error('Either source or destination must be a valid target string');
        cleanExit(1, logger);
    }

    const sourceOrDest = !! sourceParsedString ? argv.source : argv.destination;

    const parsedTarget = await disambiguateTargetName(argv.targetType, sourceOrDest, logger, dynamicConfigs, ssmTargets, sshTargets, envs);

    // Given the command level parsing only accepts ssm and ssh
    // we should not expect to his this code block
    if(parsedTarget.type == TargetType.DYNAMIC)
    {
        logger.error('Cannot file transfer with a dynamic access config');
        logger.warn('Please create a new dynamic access target or fetch an existing one (zli lt)');
        cleanExit(1, logger);
    }

    const fileService = new FileService(configService, logger);

    // figure out upload or download
    // would be undefined if not parsed properly
    if(!! destParsedString)
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

    } else if(!! sourceParsedString) {
    // download case
        await fileService.downloadFile(parsedTarget.id, parsedTarget.type, parsedTarget.path, argv.destination, parsedTarget.user);
    } else {
        logger.error('Invalid target string, must follow syntax:');
        logger.error(targetStringExample);
    }
    await cleanExit(0, logger);
}