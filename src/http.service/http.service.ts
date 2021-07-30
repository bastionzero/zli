import { IdP, TargetType } from '../types';
import got, { Got, HTTPError } from 'got/dist/source';
import { Dictionary } from 'lodash';
import { ClientSecretResponse, CloseConnectionRequest, CloseSessionRequest, CloseSessionResponse, ConnectionSummary, CreateConnectionRequest, CreateConnectionResponse, CreateSessionRequest, CreateSessionResponse, DownloadFileRequest, DynamicAccessConfigSummary, EnvironmentDetails, GetAutodiscoveryScriptRequest, GetAutodiscoveryScriptResponse, GetTargetPolicyRequest, GetTargetPolicyResponse, ListSessionsResponse, ListSsmTargetsRequest, MfaClearRequest, MfaResetRequest, MfaResetResponse, MfaTokenRequest, MixpanelTokenResponse, SessionDetails, SshTargetSummary, SsmTargetSummary, TargetUser, UploadFileRequest, UploadFileResponse, UserRegisterResponse, UserSummary, Verb, GetKubeUnregisteredAgentYamlResponse, GetKubeUnregisteredAgentYamlRequest, ClusterSummary, KubeProxyResponse, KubeProxyRequest, KubernetesPolicySummary, UpdateKubePolicyRequest, GetUserInfoResponse, GetUserInfoRequest, KubeProxyDescribeRequest, KubeProxyDescribeResponse} from './http.service.types';
import { ConfigService } from '../config.service/config.service';
import fs, { ReadStream } from 'fs';
import FormData from 'form-data';
import { Logger } from '../../src/logger.service/logger';
import { cleanExit } from '../../src/handlers/clean-exit.handler';

export class HttpService
{
    // ref for got: https://github.com/sindresorhus/got
    protected httpClient: Got;
    private baseUrl: string;
    protected configService: ConfigService;
    private authorized: boolean;
    private logger: Logger;

    constructor(configService: ConfigService, serviceRoute: string, logger: Logger, authorized: boolean = true)
    {
        this.configService = configService;
        this.authorized = authorized;
        this.logger = logger;
        this.baseUrl = `${this.configService.serviceUrl()}${serviceRoute}`;

        this.httpClient = got.extend({
            prefixUrl: this.baseUrl,
            // Remember to set headers before calling API
            hooks: {
                beforeRequest: [
                    (options) => this.logger.trace(`Making request to: ${options.url}`)
                ],
                afterResponse: [
                    (response, _) => {
                        this.logger.trace(`Request completed to: ${response.url}`);
                        return response;
                    }
                ]
            }
            // throwHttpErrors: false // potentially do this if we want to check http without exceptions
        });
    }

    private setHeaders()
    {
        const headers: Dictionary<string> = {};

        if(this.authorized) headers['Authorization'] = this.configService.getAuthHeader();
        if(this.authorized && this.configService.sessionId()) headers['X-Session-Id'] = this.configService.sessionId();

        // append headers
        this.httpClient = this.httpClient.extend({ headers: headers });
    }

    private async handleHttpException(route: string, error: HTTPError) : Promise<void>
    {
        this.logger.debug(`Error in ${this.baseUrl}${route}`);
        let errorMessage = error.message;

        if(!error.response) {
            this.logger.error(`HttpService Error:\n${errorMessage}`);
            await cleanExit(1, this.logger);
        }

        if(error.response.statusCode == 401) {
            this.logger.error(`Authentication Error:\n${errorMessage}`);
        } else {
            // Handle 500 errors by printing out our custom exception message
            if(error.response.statusCode == 500) {
                errorMessage = JSON.stringify(JSON.parse(error.response.body as string), null, 2);
            } else if(error.response.statusCode == 502)
            {
                this.logger.error('Service is offline');
            }
            this.logger.error(`HttpService Error:\n${errorMessage}`);
        }

        await cleanExit(1, this.logger);
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
            const resp : TResp = await this.httpClient.get(
                route,
                {
                    searchParams: queryParams,
                    parseJson: text => JSON.parse(text),
                }
            ).json();
            return resp;
        } catch(error) {
            this.handleHttpException(route, error);
        }
    }

    protected async Post<TReq, TResp>(route: string, body: TReq) : Promise<TResp>
    {
        this.setHeaders();

        try {
            const resp : TResp = await this.httpClient.post(
                route,
                {
                    json: body,
                    parseJson: text => JSON.parse(text),
                }
            ).json();
            return resp;
        } catch(error) {
            this.handleHttpException(route, error);
        }
    }

    protected async FormPostWithException<TReq, TResp>(route: string, body: TReq): Promise<TResp> {
        this.setHeaders();

        const formBody = this.getFormDataFromRequest(body);

        const resp : TResp = await this.httpClient.post(
            route,
            {
                body: formBody
            }
        ).json();
        return resp;
    }

    protected async FormPost<TReq, TResp>(route: string, body: TReq) : Promise<TResp>
    {
        this.setHeaders();

        const formBody = this.getFormDataFromRequest(body);

        try {
            const resp : TResp = await this.httpClient.post(
                route,
                {
                    body: formBody
                }
            ).json();
            return resp;
        } catch (error) {
            this.handleHttpException(route, error);
        }
    }

    // Returns a request object that you can add response handlers to at a higher layer
    protected FormStream<TReq>(route: string, body: TReq, localPath: string) : Promise<void>
    {
        this.setHeaders();

        const formBody = this.getFormDataFromRequest(body);
        const whereToSave = localPath.endsWith('/') ? localPath + `bzero-download-${Math.floor(Date.now() / 1000)}` : localPath;

        return new Promise((resolve, reject) => {
            try {
                const requestStream = this.httpClient.stream.post(
                    route,
                    {
                        isStream: true,
                        body: formBody
                    }
                );

                // Buffer is returned by 'data' event
                requestStream.on('data', (response: Buffer) => {
                    fs.writeFileSync(whereToSave, response);
                });

                requestStream.on('end', () => {
                    this.logger.info('File download complete');
                    this.logger.info(whereToSave);
                    resolve();
                });
            } catch (error) {
                this.handleHttpException(route, error);
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

    public ListSsmTargets(showDynamic: boolean) : Promise<SsmTargetSummary[]>
    {
        const req: ListSsmTargetsRequest = {
            showDynamicAccessTargets: showDynamic
        };

        return this.Post<ListSsmTargetsRequest, SsmTargetSummary[]>('list', req);
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

        const _ : UploadFileResponse = await this.FormPost('upload', request);

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

export class DynamicAccessConfigService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/dynamic-access/', logger);
    }

    public ListDynamicAccessConfigs(): Promise<DynamicAccessConfigSummary[]>
    {
        return this.Post('list', {});
    }
}

export class PolicyQueryService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/policy-query', logger);
    }

    public ListTargetUsers(targetId: string, targetType: TargetType, verb?: Verb, targetUser?: TargetUser): Promise<GetTargetPolicyResponse>
    {
        const request: GetTargetPolicyRequest = {
            targetId: targetId,
            targetType: targetType,
            verb: verb,
            targetUser: targetUser
        };

        return this.Post('target-connect', request);
    }

    public CheckKubeProxy(
        clusterName: string,
        clusterRole: string,
        environmentId: string,
    ): Promise<KubeProxyResponse>
    {
        const request: KubeProxyRequest = {
            clusterName: clusterName,
            clusterRole: clusterRole,
            environmentId: environmentId,
        };

        return this.FormPost('kube-proxy', request);
    }

    public DescribeKubeProxy(
        clusterName: string,
    ): Promise<KubeProxyDescribeResponse>
    {
        const request: KubeProxyDescribeRequest = {
            clusterName: clusterName,
        };

        return this.FormPost('describe-kube-proxy', request);
    }
}

export class AutoDiscoveryScriptService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/AutodiscoveryScript', logger);
    }

    public getAutodiscoveryScript(
        operatingSystem: string,
        targetName: string,
        environmentId: string,
        agentVersion: string
    ): Promise<GetAutodiscoveryScriptResponse>
    {
        const request: GetAutodiscoveryScriptRequest = {
            apiUrl: `${this.configService.serviceUrl()}api/v1/`,
            targetNameScript: `TARGET_NAME=\"${targetName}\"`,
            envId: environmentId,
            agentVersion: agentVersion
        };

        return this.Post(`${operatingSystem}`, request);
    }
}

export class KubeService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/kube', logger);
    }

    public getKubeUnregisteredAgentYaml(
        clusterName: string,
        labels: string,
        namespace: string
    ): Promise<GetKubeUnregisteredAgentYamlResponse>
    {
        const request: GetKubeUnregisteredAgentYamlRequest = {
            clusterName: clusterName,
            labels: labels,
            namespace: namespace,
        };
        return this.FormPost('get-agent-yaml', request);
    }

    public GetUserInfoFromEmail(
        email: string
    ): Promise<GetUserInfoResponse>
    {
        const request: GetUserInfoRequest = {
            email: email,
        };

        return this.FormPostWithException('get-user', request);
    }

    public ListKubeClusters(): Promise<ClusterSummary[]> {
        return this.Post('list', {});
    }
}

export class PolicyService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/Policy', logger);
    }

    public ListAllPolicies(
    ): Promise<KubernetesPolicySummary[]>
    {
        return this.Post('list', {});
    }

    public UpdateKubePolicy(
        policy: KubernetesPolicySummary
    ): Promise<void> {
        const request: UpdateKubePolicyRequest = {
            id: policy.id,
            name: policy.name,
            type: policy.type,
            subjects: policy.subjects,
            groups: policy.groups,
            context: JSON.stringify(policy.context),
            policyMetadata: policy.metadata
        }
        return this.Post('edit', request)
    }
}