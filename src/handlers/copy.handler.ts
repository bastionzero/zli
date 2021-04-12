import { FileService, PolicyQueryService } from '../http.service/http.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';
import { ParsedTargetString, TargetType } from '../types';
import { targetStringExampleNoPath } from '../utils';
import { VerbType } from '../http.service/http.service.types';

import _ from 'lodash';
import fs from 'fs';


export async function copyHandler(
    configService: ConfigService,
    logger: Logger,
    parsedTarget: ParsedTargetString,
    localFilePath: string,
    isTargetSource: boolean) {

    const fileService = new FileService(configService, logger);

    if(! parsedTarget) {
        logger.error('No targets matched your targetName/targetId or invalid target string, must follow syntax:');
        logger.error(targetStringExampleNoPath);
        await cleanExit(1, logger);
    }

    const policyQueryService = new PolicyQueryService(configService, logger);
    const response = await policyQueryService.ListTargetUsers(parsedTarget.id, parsedTarget.type, {type: VerbType.FileTransfer}, undefined);

    if(! response.allowed)
    {
        logger.error('You do not have sufficient permission to file transfer with the target');
        cleanExit(1, logger);
    }

    const allowedTargetUsers = response.allowedTargetUsers.map(u => u.userName);
    if(response.allowedTargetUsers && ! _.includes(allowedTargetUsers, parsedTarget.user)) {
        logger.error(`You do not have permission to file transfer as targetUser: ${parsedTarget.user}`);
        logger.info(`Current allowed users for you: ${allowedTargetUsers}`);
        cleanExit(1, logger);
    }

    if(parsedTarget.type == TargetType.DYNAMIC)
    {
        logger.error('Cannot file transfer with a dynamic access config');
        logger.warn('Please create a new dynamic access target or fetch an existing one (zli lt)');
        cleanExit(1, this.logger);
    }

    // figure out upload or download
    // would be undefined if not parsed properly
    if(isTargetSource)
    {
        // Upload case
        // First ensure that the file exists
        if (!fs.existsSync(localFilePath)) {
            logger.warn(`File ${localFilePath} does not exist!`);
            process.exit(1);
        }

        // Then create our read stream and try to upload it
        const fh = fs.createReadStream(localFilePath);
        await fileService.uploadFile(parsedTarget.id, parsedTarget.type, parsedTarget.path, fh, parsedTarget.user);
        logger.info('File upload complete');

    } else {
    // download case
        await fileService.downloadFile(parsedTarget.id, parsedTarget.type, parsedTarget.path, localFilePath, parsedTarget.user);
    }
    await cleanExit(0, logger);
}