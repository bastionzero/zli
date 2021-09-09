import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { GroupSummary } from './groups.types';

export class GroupsService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/groups', logger);
    }

    public ListGroups(): Promise<GroupSummary[]>
    {
        return this.Get('list', {});
    }

    public FetchGroups(): Promise<GroupSummary[]>
    {
        return this.Post('fetch', {});
    }
}