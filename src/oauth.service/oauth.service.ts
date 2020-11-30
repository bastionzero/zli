import { AuthorizationParameters, Client, custom, generators, Issuer, TokenSet, TokenSetParameters, UserinfoResponse } from "openid-client";
import open from 'open';
import { IDisposable } from "../websocket.service/websocket.service";
import http, { RequestListener } from "http";
import { setTimeout } from "timers";
import { thoumMessage } from "../cli-driver";

export class OAuthService implements IDisposable {
    private authServiceUrl: string;
    private server: http.Server; // callback listener
    private callbackPort: number;
    private host: string = '127.0.0.1';

    // TODO inject configService
    constructor(authServiceUrl: string, callbackPort: number = 3000) {
        this.authServiceUrl = authServiceUrl;
        this.callbackPort = callbackPort;
    }

    private setupCallbackListener(client: Client, codeVerifier: string, callback: (tokenSet: TokenSet, expireTime: number) => void, resolve: (value?: void | PromiseLike<void>) => void): void {

        const requestListener: RequestListener = async (req, res) => {
            res.writeHead(200);

            switch (req.url.split('?')[0]) {
                case "/login-callback":
                    const params = client.callbackParams(req);

                    const tokenSet = await client.callback(`http://${this.host}:${this.callbackPort}/login-callback`, params, { code_verifier: codeVerifier });
                    const tokenSetExpireTime: number = (Math.floor(Date.now() / 1000)) + (30); // 12 hours minus 30 seconds from now (epoch time in seconds)
                    thoumMessage(`log in successful`);
                    thoumMessage(`callback listener closed`);

                    // write to config with callback
                    callback(tokenSet, tokenSetExpireTime);
                    this.server.close();
                    res.end('Log in successful. You may close this window'); // TODO: serve HTML here
                    resolve();
                    break;

                case '/logout-callback':
                    thoumMessage(`log in successful`);
                    thoumMessage(`callback listener closed`);
                    res.end('Log out successful. You may close this window'); // TODO: serve HTML here
                    resolve();
                    break;

                default:
                    // console.log(`default callback at: ${req.url}`);
                    break;
            }
        };

        this.server = http.createServer(requestListener);
        this.server.listen(this.callbackPort, this.host, () => {});
        thoumMessage(`callback listening on http://${this.host}:${this.callbackPort}/`);
    }

    // The client will make the log-in requests with the following parameters
    private async getClient(): Promise<Client>
    {
        const clunk80Auth = await Issuer.discover(this.authServiceUrl);
        var client = new clunk80Auth.Client({
            client_id: 'CLI',
            redirect_uris: [`http://${this.host}:${this.callbackPort}/login-callback`],
            response_types: ['code'],
            token_endpoint_auth_method: 'none',
        });

        // set clock skew
        // ref: https://github.com/panva/node-openid-client/blob/77d7c30495df2df06c407741500b51498ba61a94/docs/README.md#customizing-clock-skew-tolerance
        client[custom.clock_tolerance] = 5 * 60; // 5 minute clock skew allowed for verification

        return client;
    }

    public login(callback: (tokenSet: TokenSet, expireTime: number) => void): Promise<void>
    {
        return new Promise<void>(async (resolve, reject) => {
            setTimeout(() => reject('Log in timeout reached'), 3 * 60 * 1000);

            const client = await this.getClient();
            const code_verifier = generators.codeVerifier();
            const code_challenge = generators.codeChallenge(code_verifier);

            this.setupCallbackListener(client, code_verifier, callback, resolve);

            // parameters that get serialized into the url
            var authParams: AuthorizationParameters = {
                client_id: 'CLI',
                code_challenge: code_challenge,
                code_challenge_method: 'S256',
                // both openid and offline_access must be set for refresh token
                scope: 'openid offline_access email profile backend-api',
            };

            await open(client.authorizationUrl(authParams));
        });
    }

    public async refresh(tokenSetParams: TokenSetParameters): Promise<TokenSet>
    {
        const client = await this.getClient();
        const tokenSet = new TokenSet(tokenSetParams);
        const refreshedTokenSet = await client.refresh(tokenSet);

        return refreshedTokenSet;
    }

    public async userInfo(tokenSetParams: TokenSetParameters): Promise<UserinfoResponse>
    {
        const client = await this.getClient();
        const tokenSet = new TokenSet(tokenSetParams);
        const userInfo = await client.userinfo(tokenSet);
        return userInfo;
    }

    public logout(tokenSetParams: TokenSetParameters): Promise<void>
    {
        return new Promise<void>(async (resolve, reject) => 
        {
            setTimeout(() => reject('Log out timeout reached'), 3 * 60 * 1000);

            const client = await this.getClient();
            const tokenSet = new TokenSet(tokenSetParams);
            
            // TODO: come up with better callback listener flow for login and logout flows
            this.setupCallbackListener(
                client, 
                undefined, 
                () => {}, 
                resolve
            ); 

            const endSessionUrl = client.endSessionUrl({post_logout_redirect_uri: `http://${this.host}:${this.callbackPort}/logout-callback`, id_token_hint: tokenSet});

            await open(endSessionUrl);
        });
    }

    dispose(): void {
        if(this.server)
        {
            this.server.close();
            this.server = undefined;
        }
    }
}