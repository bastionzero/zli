import { GroupsService } from '../http.service/http.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';
import { getTableOfGroups } from '../utils';

export async function fetchGroupsHandler(
    argv: any,
    configService: ConfigService,
    logger: Logger,
){
    const groupsService = new GroupsService(configService, logger);
    const groups = await groupsService.FetchGroups();
    if(!! argv.json) {
        // json output
        console.log(JSON.stringify(groups));
    } else {
        if (groups.length === 0){
            logger.info('There are no available groups');
            await cleanExit(0, logger);
        }
        // regular table output
        const tableString = getTableOfGroups(groups);
        console.log(tableString);
    }

    await cleanExit(0, logger);
}