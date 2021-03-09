import { AuthConfigService } from '../../webshell-common-ts/auth-config-service/auth-config.service';
import { ConfigService } from './config.service';

export class ZliAuthConfigService implements AuthConfigService {

    constructor(
        private configService: ConfigService
    )
    {}

    getServiceUrl() {
        return this.configService.serviceUrl() + 'api/v1/';
    }

    getSessionId() {
        return this.configService.sessionId();
    }

    async getIdToken() {
        return this.configService.getAuth();
    }
}