import { AuthConfigService } from '../../../webshell-common-ts/auth-config-service/auth-config.service';
import { ConfigService } from './config.service';
import { Logger } from '../logger/logger.service';
import { OAuthService } from '../oauth/oauth.service';

export class ZliAuthConfigService implements AuthConfigService {

    private oauth: OAuthService;

    constructor(
        private configService: ConfigService,
        private logger: Logger
    )
    {
        this.oauth = new OAuthService(this.configService, this.logger);
    }

    getServiceUrl() {
        return this.configService.serviceUrl() + 'api/v1/';
    }

    getSessionId() {
        return this.configService.sessionId();
    }

    async getIdToken() {
        return await this.oauth.getIdToken();
    }
}