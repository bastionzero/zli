import { TargetType } from '../types';
import got, { Got, HTTPError } from 'got/dist/source';
import { Dictionary } from 'lodash';
import { CloseConnectionRequest, CloseSessionRequest, CloseSessionResponse, ConnectionSummary, CreateConnectionRequest, CreateConnectionResponse, CreateSessionRequest, CreateSessionResponse, DownloadFileRequest, EnvironmentDetails, ListSessionsResponse, SessionDetails, SshTargetSummary, SsmTargetSummary, UploadFileRequest, UploadFileResponse } from './http.service.types';
import { ConfigService } from '../config.service/config.service';
import fs, { ReadStream } from 'fs';
import FormData from 'form-data';
import { thoumError, thoumMessage } from '../utils';

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
            hooks: {
                beforeRequest: [
                    (options) => thoumMessage(`Making request to: ${options.url}`) 
                ],
                afterResponse: [
                    (response, _) => {
                        thoumMessage(`Request successful to: ${response.url}`);
                        return response;
                    }
                ]
            }
            // throwHttpErrors: false // potentially do this if we want to check http without exceptions
        });
    }

    private handleHttpException(error: HTTPError) : void
    {
        let errorMessage = error.message;

        // Handle 500 errors by printing out our custom exception message
        if(error.response.statusCode == 500) {
            errorMessage = JSON.stringify(JSON.parse(error.response.body as string), null, 2);
        } else if(error.response.statusCode == 502)
        {
            thoumError('Service is offline');
        }

        thoumError(`HttpService Error:\n${errorMessage}`);
    }

    protected getFormDataFromRequest(request: any): FormData {
        return Object.keys(request).reduce((formData, key) => {
            formData.append(key, request[key]);
            return formData;
        }, new FormData());
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
                        thoumMessage('File download complete');
                        thoumMessage(whereToSave);
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
        super(configService, 'api/v1/ssm/');
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
        super(configService, 'api/v1/ssh/');
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
    constructor(configService: ConfigService)
    {
        super(configService, 'api/v1/environment/');
    }

    public ListEnvironments() : Promise<EnvironmentDetails[]>
    {
        return this.Post('list', {});
    }
}

export class FileService extends HttpService
{
    constructor(configService: ConfigService)
    {
        super(configService, 'api/v1/file/');
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