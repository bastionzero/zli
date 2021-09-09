import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { MfaTokenRequest, MfaResetResponse, MfaResetRequest, MfaClearRequest } from './mfa.mesagges';

export class MfaService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/mfa/', logger);
    }

    public SendTotp(token: string): Promise<void>
    {
        const request : MfaTokenRequest = {
            token: token
        };

        return this.Post('totp', request);
    }

    public ResetSecret(forceSetup?: boolean): Promise<MfaResetResponse>
    {
        const request: MfaResetRequest = {
            forceSetup: !!forceSetup
        };

        return this.Post('reset', request);
    }

    public ClearSecret(userId: string): Promise<void>
    {
        const request: MfaClearRequest = {
            userId: userId
        };

        return this.Post('clear', request);
    }
}