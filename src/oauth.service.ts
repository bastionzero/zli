import { AuthorizationParameters, Client, generators, Issuer, TokenSet, TokenSetParameters } from "openid-client";
import open from 'open';
import { IDisposable } from "./websocket.service/websocket.service";
import http, { RequestListener } from "http";

export class OAuthService implements IDisposable {
    private authServiceUrl: string;
    private server: http.Server; // callback listener
    public oauthFinished: Promise<void>; // acts like a task completion source

    // TODO inject configService
    constructor(authServiceUrl: string) {
        this.authServiceUrl = authServiceUrl;
    }

    private setupCallbackListener(client: Client, codeVerifier: string, callback: (tokenSet: TokenSet, expireTime: number) => void): void {
        const host = '127.0.0.1';
        const port = 3000; // TODO: read from config

        const requestListener: RequestListener = async (req, res) => {
            res.writeHead(200);
            res.end('You may close this window'); // TODO: serve HTML here

            switch (req.url.split('?')[0]) {
                case "/login-callback":
                    const params = client.callbackParams(req);
                    
                    const tokenSet = await client.callback('http://127.0.0.1:3000/login-callback', params, { code_verifier: codeVerifier });
                    const tokenSetExpireTime: number = (Date.now() / 1000) + (60 * 60 * 12) - 30; // 12 hours minus 30 seconds from now (epoch time in seconds)
                    console.log('Tokens received and validated');
                    
                    // write to config with callback
                    callback(tokenSet, tokenSetExpireTime);

                    this.oauthFinished = Promise.resolve();
                    break;

                case '/logout-callback':
                    console.log('logout callback');
                    break;

                default:
                    console.log(`default callback at: ${req.url}`);
                    break;
            }
        };

        this.server = http.createServer(requestListener);
        this.server.listen(port, host, () => {
        });
    }

    // The client will make the log-in requests with the following parameters
    private async getClient(): Promise<Client>
    {
        const clunk80Auth = await Issuer.discover(this.authServiceUrl);
        return new clunk80Auth.Client({
            client_id: 'CLI',
            redirect_uris: ['http://127.0.0.1:3000/login-callback'],
            response_types: ['code'],
            token_endpoint_auth_method: 'none'
        });
    }

    public async login(callback: (tokenSet: TokenSet, expireTime: number) => void): Promise<void> 
    {
        this.oauthFinished = new Promise((reject, resolve) => {});
        const client = await this.getClient();
        const code_verifier = generators.codeVerifier();
        const code_challenge = generators.codeChallenge(code_verifier);

        this.setupCallbackListener(client, code_verifier, callback);

        // parameters that get serialized into the url
        var authParams: AuthorizationParameters = {
            client_id: 'CLI',
            code_challenge: code_challenge,
            code_challenge_method: 'S256',
            // both openid and offline_access must be set for refresh token
            scope: 'openid offline_access email profile backend-api',
        };

        await open(client.authorizationUrl(authParams));
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
            this.server = undefined;

        if(! this.oauthFinished)
            this.oauthFinished = Promise.resolve();
    }
}