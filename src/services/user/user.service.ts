import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { UserRegisterResponse } from './user.messages';
import { UserSummary } from './user.types';

export class UserService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/user/', logger);
    }

    public Register(): Promise<UserRegisterResponse>
    {
        return this.Post('register', {});
    }

    public Me(): Promise<UserSummary>
    {
        return this.Get('me', {});
    }

    public ListUsers(): Promise<UserSummary[]>
    {
        return this.Post('list', {});
    }
}