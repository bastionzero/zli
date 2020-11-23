import { AuthorizationParameters, Client, generators, Issuer, TokenSet, TokenSetParameters } from "openid-client";
import open from 'open';
import { IDisposable } from "./websocket.service/websocket.service";
import http, { RequestListener } from "http";
import { setTimeout } from "timers";
import chalk from "chalk";

export class OAuthService implements IDisposable {
    private authServiceUrl: string;
    private server: http.Server; // callback listener
    public oauthFinished: Promise<void>; // acts like a task completion source
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
            res.end('You may close this window'); // TODO: serve HTML here

            switch (req.url.split('?')[0]) {
                case "/login-callback":
                    const params = client.callbackParams(req);
                    
                    const tokenSet = await client.callback(`http://${this.host}:${this.callbackPort}/login-callback`, params, { code_verifier: codeVerifier });
                    const tokenSetExpireTime: number = (Date.now() / 1000) + (60 * 60 * 12) - 30; // 12 hours minus 30 seconds from now (epoch time in seconds)
                    console.log(chalk.magenta(`thoum >>> log in successful`));
                    console.log(chalk.magenta(`thoum >>> callback listener closed`));
                    
                    // write to config with callback
                    callback(tokenSet, tokenSetExpireTime);
                    this.server.close();

                    // resolve oauthFinished here
                    resolve();
                    break;

                case '/logout-callback':
                    // console.log('logout callback');
                    break;

                default:
                    // console.log(`default callback at: ${req.url}`);
                    break;
            }
        };

        this.server = http.createServer(requestListener);
        this.server.listen(this.callbackPort, this.host, () => {});
        console.log(chalk.magenta(`thoum >>> callback listening on http://${this.host}:${this.callbackPort}/`));
    }

    // The client will make the log-in requests with the following parameters
    private async getClient(): Promise<Client>
    {
        const clunk80Auth = await Issuer.discover(this.authServiceUrl);
        return new clunk80Auth.Client({
            client_id: 'CLI',
            redirect_uris: [`http://${this.host}:${this.callbackPort}/login-callback`],
            response_types: ['code'],
            token_endpoint_auth_method: 'none'
        });
    }

    public login(callback: (tokenSet: TokenSet, expireTime: number) => void): void 
    {
        this.oauthFinished = new Promise(async (resolve, reject) => {
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

    dispose(): void {
        if(this.server)
        {
            this.server.close();
            this.server = undefined;
        }

        if(! this.oauthFinished)
            this.oauthFinished = Promise.resolve();
    }
}