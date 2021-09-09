import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { ListSsmTargetsRequest } from './ssm-target.messages';
import { SsmTargetSummary } from './ssm-target.types';

export class SsmTargetService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/ssm/', logger);
    }

    public GetSsmTarget(targetId: string) : Promise<SsmTargetSummary>
    {
        return this.Get('', {id: targetId});
    }

    public ListSsmTargets(showDynamic: boolean) : Promise<SsmTargetSummary[]>
    {
        const req: ListSsmTargetsRequest = {
            showDynamicAccessTargets: showDynamic
        };

        return this.Post<ListSsmTargetsRequest, SsmTargetSummary[]>('list', req);
    }
}