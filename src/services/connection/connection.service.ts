import { TargetType } from '../common.types';
import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { CreateConnectionRequest, CreateConnectionResponse, CloseConnectionRequest } from './connection.messages';
import { ConnectionSummary, ShellConnectionAuthDetails } from './connection.types';

export class ConnectionService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/connection/', logger);
    }

    public GetConnection(connectionId: string) : Promise<ConnectionSummary>
    {
        return this.Get('', {id: connectionId});
    }

    public async CreateConnection(targetType: TargetType, targetId: string, sessionId: string, targetUser: string) : Promise<string>
    {
        const req : CreateConnectionRequest = {
            serverType: targetType,
            serverId: targetId,
            sessionId: sessionId,
            username: targetUser
        };

        const resp = await this.Post<CreateConnectionRequest, CreateConnectionResponse>('create', req);

        return resp.connectionId;
    }

    public CloseConnection(connectionId: string) : Promise<void>
    {
        const req : CloseConnectionRequest = {
            connectionId: connectionId
        };

        return this.Post('close', req);
    }

    public async GetShellConnectionAuthDetails(connectionId: string) : Promise<ShellConnectionAuthDetails>
    {
        return this.Get('auth-details', {id: connectionId});
    }
}