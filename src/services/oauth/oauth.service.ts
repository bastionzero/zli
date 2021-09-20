import { AuthorizationParameters, Client, custom, errors, generators, Issuer, TokenSet, UserinfoResponse } from 'openid-client';
import open from 'open';
import { IDisposable } from '../../../webshell-common-ts/utility/disposable';
import { IdentityProvider } from '../../../webshell-common-ts/auth-service/auth.types';
import { ConfigService } from '../config/config.service';
import http, { RequestListener } from 'http';
import { setTimeout } from 'timers';
import { Logger } from '../logger/logger.service';
import { loginHtml } from './templates/login';
import { logoutHtml } from './templates/logout';
import { cleanExit } from '../../handlers/clean-exit.handler';
import { parse as QueryStringParse } from 'query-string';
import { parseIdpType } from '../../utils';

export class OAuthService implements IDisposable {
    private server: http.Server; // callback listener
    private host: string = 'localhost';
    private logger: Logger;
    private oidcClient: Client;
    private codeVerifier: string;
    private nonce: string;

    constructor(private configService: ConfigService, logger: Logger) {
        this.logger = logger;
    }

    private setupCallbackListener(
        callback: (tokenSet: TokenSet) => void,
        onListen: () => void,
        resolve: (value?: void | PromiseLike<void>) => void
    ): void {

        const requestListener: RequestListener = async (req, res) => {
            // Example of request url string
            // /login-callback?param=...
            const urlParts = req.url.split('?');
            const queryParams = QueryStringParse(urlParts[1]);

            // example of failed login attempt
            // http://localhost:3000/login-callback?error=consent_required&error_description=AADSTS65004%3a+User+decline...
            if(!! queryParams.error)
            {
                this.logger.error('User login failed: ' + queryParams.error);
                this.logger.info('Please try logging in again');
                await cleanExit(1, this.logger);
            }

            switch (urlParts[0]) {
            case '/webapp-callback':

                // Prepare config for a new login
                const provider = parseIdpType(queryParams.idp as IdentityProvider);

                if(provider === undefined) {
                    this.logger.error('The selected identity provider is not currently supported.');
                    await cleanExit(1, this.logger);
                }
                await this.configService.loginSetup(provider);

                // Setup the oidc client for a new login
                await this.setupClient();
                this.codeVerifier = generators.codeVerifier();
                const code_challenge = generators.codeChallenge(this.codeVerifier);

                // Redirect to the idp
                res.writeHead(302, {
                    'Access-Control-Allow-Origin': '*',
                    'content-type': 'text/html',
                    'Location': this.getAuthUrl(code_challenge)
                });
                res.end();

                break;

            case '/login-callback':
                if(this.oidcClient === undefined){
                    throw new Error('Unable to parse idp response with undefined OIDC client');
                }

                if(this.codeVerifier === undefined){
                    throw new Error('Unable to parse idp response with undefined code verifier');
                }

                if(this.nonce === undefined){
                    throw new Error('Unable to parse idp response with undefined nonce');
                }

                const params = this.oidcClient.callbackParams(req);

                const tokenSet = await this.oidcClient.callback(
                    `http://${this.host}:${this.configService.callbackListenerPort()}/login-callback`,
                    params,
                    { code_verifier: this.codeVerifier, nonce: this.nonce });

                this.logger.info('Login successful');
                this.logger.debug('callback listener closed');

                // write to config with callback
                callback(tokenSet);
                this.server.close();
                res.writeHead(200, {
                    'Access-Control-Allow-Origin': '*',
                    'content-type': 'text/html'
                });
                res.write(loginHtml);
                resolve();
                break;

            case '/logout-callback':
                this.logger.info('Login successful');
                this.logger.debug('callback listener closed');
                this.server.close();
                res.write(logoutHtml);
                resolve();
                break;

            default:
                this.logger.debug(`Unhandled callback at: ${req.url}`);
                break;
            }
        };

        this.logger.debug(`Setting up callback listener at http://${this.host}:${this.configService.callbackListenerPort()}/`);
        this.server = http.createServer(requestListener);
        // Port binding failure will produce error event
        this.server.on('error', async () => {
            this.logger.error('Log in listener could not bind to port');
            this.logger.warn(`Please make sure port ${this.configService.callbackListenerPort()} is open/whitelisted`);
            this.logger.warn('To edit callback port please run: \'zli config\'');
            await cleanExit(1, this.logger);
        });
        // open browser after successful port binding
        this.server.on('listening', onListen);
        this.server.listen(this.configService.callbackListenerPort(), this.host, () => {});
    }

    // The client will make the log-in requests with the following parameters
    private async setupClient(): Promise<void>
    {
        // If the client has already been set
        if(this.oidcClient !== undefined)
            return;
        const authority = await Issuer.discover(this.configService.authUrl());
        this.oidcClient = new authority.Client({
            client_id: this.configService.clientId(),
            redirect_uris: [`http://${this.host}:${this.configService.callbackListenerPort()}/login-callback`],
            response_types: ['code'],
            token_endpoint_auth_method: 'client_secret_basic',
            client_secret: this.configService.clientSecret()
        });

        // set clock skew
        // ref: https://github.com/panva/node-openid-client/blob/77d7c30495df2df06c407741500b51498ba61a94/docs/README.md#customizing-clock-skew-tolerance
        this.oidcClient[custom.clock_tolerance] = 5 * 60; // 5 minute clock skew allowed for verification
    }

    private getAuthUrl(code_challenge: string) : string
    {
        if(this.oidcClient === undefined){
            throw new Error('Unable to get authUrl from undefined OIDC client');
        }

        if(this.nonce === undefined){
            throw new Error('Unable to get authUrl from with undefined nonce');
        }

        const authParams: AuthorizationParameters = {
            client_id: this.configService.clientId(), // This one gets put in the queryParams
            response_type: 'code',
            code_challenge: code_challenge,
            code_challenge_method: 'S256',
            scope: this.configService.authScopes(),
            // required for google refresh token
            prompt: 'consent',
            access_type: 'offline',
            nonce: this.nonce
        };

        return this.oidcClient.authorizationUrl(authParams);
    }

    public isAuthenticated(): boolean
    {
        const tokenSet = this.configService.tokenSet();

        if(tokenSet === undefined)
            return false;

        return !tokenSet.expired();
    }

    public login(callback: (tokenSet: TokenSet) => void, nonce?: string): Promise<void>
    {
        this.nonce = nonce;
        return new Promise<void>(async (resolve, reject) => {
            setTimeout(() => reject(this.logger.error('Login timeout reached')), 60 * 1000);

            const openBrowser = async () => await open(`${this.configService.serviceUrl()}authentication/login?zliLogin=true`);

            this.setupCallbackListener(callback, openBrowser, resolve);
        });
    }

    public async refresh(): Promise<TokenSet>
    {
        await this.setupClient();
        const tokenSet = this.configService.tokenSet();
        const refreshToken = tokenSet.refresh_token;
        const refreshedTokenSet = await this.oidcClient.refresh(tokenSet);

        // In case of google the refreshed token is not returned in the refresh
        // response so we set it from the previous value
        if(! refreshedTokenSet.refresh_token)
            refreshedTokenSet.refresh_token = refreshToken;

        return refreshedTokenSet;
    }

    // If you need the token.sub this is where you can get it
    public async userInfo(): Promise<UserinfoResponse>
    {
        await this.setupClient();
        const tokenSet = this.configService.tokenSet();
        const userInfo = await this.oidcClient.userinfo(tokenSet);
        return userInfo;
    }

    // Returns the current OAuth idtoken. Refreshes it before returning if expired
    public async getIdToken(): Promise<string> {

        const tokenSet = this.configService.tokenSet();

        // decide if we need to refresh or prompt user for login
        if(tokenSet)
        {
            if(this.configService.tokenSet().expired())
            {
                try {
                    this.logger.debug('Refreshing oauth token');

                    const newTokenSet = await this.refresh();
                    this.configService.setTokenSet(newTokenSet);
                    this.logger.debug('Oauth token refreshed');
                } catch(e) {
                    if(e instanceof errors.RPError || e instanceof errors.OPError) {
                        this.logger.error('Stale log in detected');
                        this.logger.info('You need to log in, please run \'zli login --help\'');
                        this.configService.logout();
                        await cleanExit(1, this.logger);
                    } else {
                        this.logger.error('Unexpected error during oauth refresh');
                        this.logger.info('Please log in again');
                        this.configService.logout();
                        await cleanExit(1, this.logger);
                    }
                }
            }
        } else {
            this.logger.error('You need to log in, please run \'zli login --help\'');
            await cleanExit(1, this.logger);
        }

        return this.configService.getAuth();
    }

    dispose(): void {
        if(this.server)
        {
            this.server.close();
            this.server = undefined;
        }
    }
}