import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { DynamicAccessConfigSummary } from './dynamic-access-config.types';

export class DynamicAccessConfigService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/dynamic-access/', logger);
    }

    public ListDynamicAccessConfigs(): Promise<DynamicAccessConfigSummary[]>
    {
        return this.Post('list', {});
    }
}