import { TargetType } from '../types';
import got, { Got, HTTPError } from 'got/dist/source';
import { Dictionary } from 'lodash';
import { CloseConnectionRequest, CloseSessionRequest, CloseSessionResponse, ConnectionSummary, CreateConnectionRequest, CreateConnectionResponse, CreateSessionRequest, CreateSessionResponse, ListSessionsResponse, SessionDetails, SshTargetSummary, SsmTargetSummary } from './http.service.types';
import { ConfigService } from '../config.service/config.service';
import chalk from 'chalk';

export class HttpService
{
    // ref for got: https://github.com/sindresorhus/got
    protected httpClient: Got;
    private configService: ConfigService;

    constructor(configService: ConfigService, serviceRoute: string)
    {
        this.configService = configService;

        this.httpClient = got.extend({
            prefixUrl: `${this.configService.serviceUrl()}${serviceRoute}`,
            headers: {authorization: this.configService.getAuthHeader()},
            // throwHttpErrors: false // potentially do this if we want to check http without exceptions
        });
    }

    private handleHttpException(reason?: any) : void 
    {
        console.log(chalk.red(`HttpService Error:\n${reason}`));
    }

    protected async Get<TResp>(route: string, queryParams: Dictionary<string>) : Promise<TResp>
    {
        try {
            var resp : TResp = await this.httpClient.get(
                route,
                {
                    searchParams: queryParams,
                    parseJson: text => JSON.parse(text),
                }
            ).json();    
        } catch(error) {
            this.handleHttpException(error)
        }
        
        return resp;
    }

    protected async Post<TReq, TResp>(route: string, body: TReq) : Promise<TResp>
    {
        try{
            var resp : TResp = await this.httpClient.post(
                route,
                {
                    json: body,
                    parseJson: text => JSON.parse(text)
                }
            ).json();
        } catch(error) {
            this.handleHttpException(error);
        }

        return resp;
    }
}

export class SessionService extends HttpService
{
    constructor(configService: ConfigService)
    {
        super(configService, 'api/v1/session/');
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
        var req : CreateSessionRequest = {connectionsToOpen: []};
        
        if(sessionName)
            req.displayName = sessionName;

        const resp = await this.Post<CreateSessionRequest, CreateSessionResponse>('create', req);

        return resp.sessionId;
    }

    public CloseSession(sessionId: string) : Promise<CloseSessionResponse>
    {
        var req : CloseSessionRequest = {sessionId: sessionId}
        return this.Post('close', req);
    }
}

export class ConnectionService extends HttpService
{
    constructor(configService: ConfigService)
    {
        super(configService, 'api/v1/connection/');
    }

    public GetConnection(connectionId: string) : Promise<ConnectionSummary>
    {
        return this.Get('', {id: connectionId});
    }

    public async CreateConnection(targetType: TargetType, targetId: string, sessionId: string, targetUser: string) : Promise<string>
    {
        var req : CreateConnectionRequest = {
            serverType: targetType, 
            serverId: targetId, 
            sessionId: sessionId,
            username: targetUser
        };
        
        const resp = await this.Post<CreateConnectionRequest, CreateConnectionResponse>('create', req);

        return resp.connectionId;
    }

    public CloseConnection(connectionId: string) : Promise<any>
    {
        var req : CloseConnectionRequest = {
            connectionId: connectionId
        };

        return this.Post('close', req);
    }
}

export class SsmTargetService extends HttpService
{
    constructor(configService: ConfigService)
    {
        super(configService, 'api/v1/ssmTarget/');
    }

    public GetSsmTarget(targetId: string) : Promise<SsmTargetSummary>
    {
        return this.Get('', {id: targetId});
    }

    public ListSsmTargets() : Promise<SsmTargetSummary[]>
    {
        return this.Post('list', {});
    }
}

export class SshTargetService extends HttpService
{
    constructor(configService: ConfigService)
    {
        super(configService, 'api/v1/sshTarget/');
    }

    public GetSsmTarget(targetId: string) : Promise<SshTargetSummary>
    {
        return this.Get('', {id: targetId});
    }

    public ListSsmTargets() : Promise<SshTargetSummary[]>
    {
        return this.Post('list', {});
    }
}