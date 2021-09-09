import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { ListSessionsResponse, CreateSessionRequest, CreateSessionResponse, CloseSessionResponse, CloseSessionRequest } from './session.messages';
import { SessionDetails } from './session.types';

export class SessionService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/session/', logger);
    }

    public GetSession(sessionId: string) : Promise<SessionDetails>
    {
        return this.Get('', {id: sessionId});
    }

    public ListSessions() : Promise<ListSessionsResponse>
    {
        return this.Post('list', {});
    }

    public async CreateSession(sessionName? : string) : Promise<string>
    {
        const req : CreateSessionRequest = {connectionsToOpen: []};

        if(sessionName)
            req.displayName = sessionName;

        const resp = await this.Post<CreateSessionRequest, CreateSessionResponse>('create', req);

        return resp.sessionId;
    }

    public CloseSession(sessionId: string) : Promise<CloseSessionResponse>
    {
        const req : CloseSessionRequest = {sessionId: sessionId};
        return this.Post('close', req);
    }
}