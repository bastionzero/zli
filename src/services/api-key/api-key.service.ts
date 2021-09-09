import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { ApiKeyDetails } from './api-key.types';

export class ApiKeyService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/ApiKey', logger);
    }

    public ListAllApiKeys(): Promise<ApiKeyDetails[]>
    {
        return this.Post('list', {});
    }
}