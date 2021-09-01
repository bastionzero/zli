import { UserService } from '../http.service/http.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';
import { getTableOfUsers } from '../utils';

export async function listUsersHandler(
    argv: any,
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