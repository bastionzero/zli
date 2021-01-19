import { AuthorizationParameters, Client, custom, generators, Issuer, TokenSet, TokenSetParameters, UserinfoResponse } from "openid-client";
import open from 'open';
import { IDisposable } from "../websocket.service/websocket.service";
import { ConfigService } from "../config.service/config.service";
import http, { RequestListener } from "http";
import { setTimeout } from "timers";
import { Logger } from "../../src/logger.service/logger";

export class OAuthService implements IDisposable {
    private server: http.Server; // callback listener
    private host: string = 'localhost';
    private logger: Logger;

    constructor(private configService: ConfigService, logger: Logger) {
        this.logger = logger;
    }

    private setupCallbackListener(
        client: Client, 
        codeVerifier: string, 
        callback: (tokenSet: TokenSet) => void,
        onListen: () => void,
        resolve: (value?: void | PromiseLike<void>) => void
    ): void {

        const requestListener: RequestListener = async (req, res) => {
            res.writeHead(200);

            switch (req.url.split('?')[0]) {
                case "/login-callback":
                    const params = client.callbackParams(req);

                    const tokenSet = await client.callback(`http://${this.host}:${this.configService.callbackListenerPort()}/login-callback`, params, { code_verifier: codeVerifier });

                    this.logger.info(`Login successful`);
                    this.logger.debug(`callback listener closed`);

                    // write to config with callback
                    callback(tokenSet);
                    this.server.close();
                    res.end('Log in successful. You may close this window.'); // TODO: serve HTML here
                    resolve();
                    break;

                case '/logout-callback':
                    this.logger.info(`Login successful`);
                    this.logger.debug(`callback listener closed`);
                    res.end('Log out successful. You may close this window.'); // TODO: serve HTML here
                    resolve();
                    break;

                default:
                    // console.log(`default callback at: ${req.url}`);
                    break;
            }
        };

        this.logger.debug(`Setting up callback listener at http://${this.host}:${this.configService.callbackListenerPort()}/`);
        this.server = http.createServer(requestListener);
        // Port binding failure will produce error event
        this.server.on('error', () => {
            this.logger.error('Log in listener could not bind to port');
            this.logger.warn(`Please make sure port ${this.configService.callbackListenerPort()} is open/whitelisted`);
            this.logger.warn('To edit callback port please run: \'thoum config\'');
            process.exit(1);
        });
        // open browser after successful port binding
        this.server.on('listening', onListen);
        this.server.listen(this.configService.callbackListenerPort(), this.host, () => {});
    }

    // The client will make the log-in requests with the following parameters
    private async getClient(): Promise<Client>
    {
        const clunk80Auth = await Issuer.discover(this.configService.authUrl());
        var client = new clunk80Auth.Client({
            client_id: this.configService.clientId(),
            redirect_uris: [`http://${this.host}:${this.configService.callbackListenerPort()}/login-callback`],
            response_types: ['code'],
            token_endpoint_auth_method: 'client_secret_basic',
            client_secret: this.configService.clientSecret()
        });

        // set clock skew
        // ref: https://github.com/panva/node-openid-client/blob/77d7c30495df2df06c407741500b51498ba61a94/docs/README.md#customizing-clock-skew-tolerance
        client[custom.clock_tolerance] = 5 * 60; // 5 minute clock skew allowed for verification

        return client;
    }

    private getAuthUrl(client: Client, code_challenge: string) : string
    {
        const authParams: AuthorizationParameters = {
            client_id: this.configService.clientId(), // This one gets put in the queryParams
            response_type: 'code',
            code_challenge: code_challenge,
            code_challenge_method: 'S256',
            scope: this.configService.authScopes(),
            // required for google refresh token
            prompt: 'consent',
            access_type: 'offline'
        };

        return client.authorizationUrl(authParams);
    }

    public isAuthenticated(): boolean
    {
        const tokenSet = this.configService.tokenSet();

        if(tokenSet === undefined)
            return false;

        return tokenSet.expired();
    }

    public login(callback: (tokenSet: TokenSet) => void): Promise<void>
    {
        return new Promise<void>(async (resolve, reject) => {
            setTimeout(() => reject('Log in timeout reached'), 60 * 1000);

            const client = await this.getClient();
            const code_verifier = generators.codeVerifier();
            const code_challenge = generators.codeChallenge(code_verifier);

            const openBrowser = async () => await open(this.getAuthUrl(client, code_challenge));
            
            this.setupCallbackListener(client, code_verifier, callback, openBrowser, resolve);
        });
    }

    public async refresh(): Promise<TokenSet>
    {
        const client = await this.getClient();
        const tokenSet = this.configService.tokenSet();
        const refreshToken = tokenSet.refresh_token;
        const refreshedTokenSet = await client.refresh(tokenSet);
        
        // In case of google the refreshed token is not returned in the refresh
        // response so we set it from the previous value
        if(! refreshedTokenSet.refresh_token)
            refreshedTokenSet.refresh_token = refreshToken;

        return refreshedTokenSet;
    }

    public async userInfo(): Promise<UserinfoResponse>
    {
        const client = await this.getClient();
        const tokenSet = this.configService.tokenSet();
        const userInfo = await client.userinfo(tokenSet);
        return userInfo;
    }

    dispose(): void {
        if(this.server)
        {
            this.server.close();
            this.server = undefined;
        }
    }
}