import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';
import { getTableOfGroups } from '../../utils';
import { GroupsService } from '../../services/groups/groups.service';
import yargs from 'yargs';
import { groupArgs } from './group.command-builder';

export async function fetchGroupsHandler(
    argv: yargs.Arguments<groupArgs>,
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