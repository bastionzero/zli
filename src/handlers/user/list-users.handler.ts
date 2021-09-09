import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';
import { getTableOfUsers } from '../../utils';
import { UserService } from '../../services/user/user.service';
import yargs from 'yargs';
import { userArgs } from './user.command-builder';

export async function listUsersHandler(
    argv: yargs.Arguments<userArgs>,
    configService: ConfigService,
    logger: Logger,
){
    const userService = new UserService(configService, logger);
    const users = await userService.ListUsers();
    if(!! argv.json) {
        // json output
        console.log(JSON.stringify(users));
    } else {
        if (users.length === 0){
            logger.info('There are no available users');
            await cleanExit(0, logger);
        }
        // regular table output
        const tableString = getTableOfUsers(users);
        console.log(tableString);
    }

    await cleanExit(0, logger);
}