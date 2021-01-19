import { IdP, TargetType } from '../types';
import got, { Got, HTTPError } from 'got/dist/source';
import { Dictionary } from 'lodash';
import { ClientSecretResponse, CloseConnectionRequest, CloseSessionRequest, CloseSessionResponse, ConnectionSummary, CreateConnectionRequest, CreateConnectionResponse, CreateSessionRequest, CreateSessionResponse, DownloadFileRequest, EnvironmentDetails, ListSessionsResponse, MfaClearRequest, MfaResetResponse, MfaTokenRequest, MixpanelTokenResponse, SessionDetails, SshTargetSummary, SsmTargetSummary, UploadFileRequest, UploadFileResponse, UserRegisterResponse, UserSummary } from './http.service.types';
import { ConfigService } from '../config.service/config.service';
import fs, { ReadStream } from 'fs';
import FormData from 'form-data';
import { Logger } from '../../src/logger.service/logger';

export class HttpService
{
    // ref for got: https://github.com/sindresorhus/got
    protected httpClient: Got;
    private configService: ConfigService;
    private authorized: boolean;
    private logger: Logger;

    constructor(configService: ConfigService, serviceRoute: string, logger: Logger, authorized: boolean = true)
    {
        this.configService = configService;
        this.authorized = authorized;
        this.logger = logger;

        this.httpClient = got.extend({
            prefixUrl: `${this.configService.serviceUrl()}${serviceRoute}`,
            // Remember to set headers before calling API
            hooks: {
                beforeRequest: [
                    (options) => this.logger.debug(`Making request to: ${options.url}`) 
                ],
                afterResponse: [
                    (response, _) => {
                        this.logger.debug(`Request completed to: ${response.url}`);
                        return response;
                    }
                ]
            }
            // throwHttpErrors: false // potentially do this if we want to check http without exceptions
        });
    }

    private setHeaders()
    {
        let headers: Dictionary<string> = {};

        if(this.authorized) headers['Authorization'] = this.configService.getAuthHeader();
        if(this.authorized && this.configService.sessionId()) headers['X-Session-Id'] = this.configService.sessionId();
        
        // append headers
        this.httpClient = this.httpClient.extend({ headers: headers });
    }

    private handleHttpException(error: HTTPError) : void
    {
        let errorMessage = error.message;
        
        if(error.response.statusCode == 401) {
            this.logger.error(`Authentication Error ${error.response.headers}`);
        }

        // Handle 500 errors by printing out our custom exception message
        if(error.response.statusCode == 500) {
            errorMessage = JSON.stringify(JSON.parse(error.response.body as string), null, 2);
        } else if(error.response.statusCode == 502)
        {
            this.logger.error('Service is offline');
        }

        this.logger.error(`HttpService Error:\n${errorMessage}`);
    }

    protected getFormDataFromRequest(request: any): FormData {
        return Object.keys(request).reduce((formData, key) => {
            formData.append(key, request[key]);
            return formData;
        }, new FormData());
    }

    protected async Get<TResp>(route: string, queryParams: Dictionary<string>) : Promise<TResp>
    {
        this.setHeaders();

        try {
            var resp : TResp = await this.httpClient.get(
                route,
                {
                    searchParams: queryParams,
                    parseJson: text => JSON.parse(text),
                }
            ).json();
        } catch(error) {
            this.handleHttpException(error);
        }

        return resp;
    }

    protected async Post<TReq, TResp>(route: string, body: TReq) : Promise<TResp>
    {
        this.setHeaders();

        try {
            var resp : TResp = await this.httpClient.post(
                route,
                {
                    json: body,
                    parseJson: text => JSON.parse(text),
                }
            ).json();
        } catch(error) {
            this.handleHttpException(error);
        }

        return resp;
    }

    protected async FormPost<TReq, TResp>(route: string, body: TReq) : Promise<TResp>
    {
        const formBody = this.getFormDataFromRequest(body);
        
        try {
            var resp : TResp = await this.httpClient.post(
                route, 
                {
                    body: formBody
                }
            ).json();
        } catch (error) {
            this.handleHttpException(error);
        }

        return resp;
    }

    // Returns a request object that you can add response handlers to at a higher layer
    protected FormStream<TReq>(route: string, body: TReq, localPath: string) : Promise<void>
    {
        const formBody = this.getFormDataFromRequest(body);
        const whereToSave = localPath.endsWith('/') ? localPath + `thoum-download-${Math.floor(Date.now() / 1000)}` : localPath;

        // TODO: bring the handlers up a level by returning the request stream to the caller?

            /* Example headers response
            {
                'content-type': 'application/octet-stream',
                'content-length': '17',
                connection: 'close',
                date: 'Tue, 01 Dec 2020 18:37:13 GMT',
                server: 'Kestrel',
                'strict-transport-security': 'max-age=2592000',
                'content-disposition': "attachment; filename=test.txt; filename*=UTF-8''test.txt",
                'x-robots-tag': 'none',
                'x-cache': 'Miss from cloudfront',
                via: '1.1 e6fc68fd040718147cda2e3ef6f63637.cloudfront.net (CloudFront)',
                'x-amz-cf-pop': 'EWR50-C1',
                'x-amz-cf-id': '4kpHjhXXK3Erk91ApPrr9Lvt9EEzCP5EjtHdkqbPobXhA9dPdxqv6g=='
            }
            */
            // TODO: something with these headers?
            // TODO: read filename from header and save if not specified
            // requestStream.on('response', (response) => {
            //     console.log(response.headers);
            // });

            return new Promise((resolve, reject) => {
                try {
                    let requestStream = this.httpClient.stream.post(
                        route,
                        {
                            isStream: true,
                            body: formBody
                        }
                    );

                    // Buffer is returned by 'data' event
                    requestStream.on('data', (response: Buffer) => {
                        fs.writeFile(whereToSave, response, () => {});
                    })
            
                    requestStream.on('end', () => {
                        this.logger.info('File download complete');
                        this.logger.info(whereToSave);
                        resolve();
                    });
                } catch (error) {
                    this.handleHttpException(error);
                    reject(error);
                }
            });
    }   
}

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
        var req : CreateConnectionRequest = {
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
        var req : CloseConnectionRequest = {
            connectionId: connectionId
        };

        return this.Post('close', req);
    }
}

export class SsmTargetService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/ssm/', logger);
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
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/ssh/', logger);
    }

    public GetSshTarget(targetId: string) : Promise<SshTargetSummary>
    {
        return this.Get('', {id: targetId});
    }

    public ListSshTargets() : Promise<SshTargetSummary[]>
    {
        return this.Post('list', {});
    }
}


export class EnvironmentService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/environment/', logger);
    }

    public ListEnvironments() : Promise<EnvironmentDetails[]>
    {
        return this.Post('list', {});
    }
}

export class FileService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/file/', logger);
    }

    public async uploadFile(targetId: string, targetType: TargetType, path: string, file: ReadStream, targetUser?: string): Promise<void> {
        const request : UploadFileRequest = {
            targetId: targetId,
            targetType: targetType,
            targetFilePath: path,
            file: file,
            targetUser: targetUser
        };
        
        const resp : UploadFileResponse = await this.FormPost('upload', request);
    
        return;
    }

    public async downloadFile(targetId: string, targetType: TargetType, targetPath: string,localPath: string, targetUser?: string): Promise<any> {
        
        const request: DownloadFileRequest = {
            targetId: targetId,
            targetType: targetType,
            filePath: targetPath,
            targetUser: targetUser
        };
    
        await this.FormStream('download', request, localPath);

        return;
    }
}
export class TokenService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/token/', logger, false);
    }

    public GetMixpanelToken(): Promise<MixpanelTokenResponse>
    {
        return this.Get('mixpanel-token', {});
    }

    public GetClientSecret(idp: IdP) : Promise<ClientSecretResponse>
    {
        return this.Get(`${idp.toLowerCase()}-client`, {});
    }
}

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

    public ResetSecret(): Promise<MfaResetResponse>
    {   
        return this.Post('reset', {});
    }

    public ClearSecret(userId: string): Promise<void>
    {
        const request: MfaClearRequest = {
            userId: userId
        };

        return this.Post('clear', request);
    }
}

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
}