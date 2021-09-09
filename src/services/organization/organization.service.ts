import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { IdentityProviderGroupsMetadataResponse } from './organization.messages';

export class OrganizationService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/organization/', logger);
    }

    public GetCredentialsMetadata(): Promise<IdentityProviderGroupsMetadataResponse>
    {
        return this.Get('groups/credentials', {});
    }
}