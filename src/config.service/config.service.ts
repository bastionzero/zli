import Conf from 'conf/dist/source';
import { TokenSet, TokenSetParameters } from 'openid-client';
import { IdP } from '../types';
import { thoumError, thoumWarn } from '../utils';

// refL: https://github.com/sindresorhus/conf/blob/master/test/index.test-d.ts#L5-L14
type ThoumConfigSchema = {
    authUrl: string,
    clientId: string,
    serviceUrl: string,
    tokenSet: TokenSetParameters,
    tokenSetExpireTime: number, // TODO: remove this when federated id server is gone
    callbackListenerPort: number,
    mixpanelToken: string,
    idp: string // Google or Microsoft
}

export class ConfigService {
    private config: Conf<ThoumConfigSchema>;

    constructor(configName: string) {
        var appName = this.getAppName(configName);
        this.config = new Conf<ThoumConfigSchema>({
            projectName: 'thoum-cli',
            configName: configName, // prod, stage, dev
            defaults: {
                authUrl: undefined,
                clientId: undefined,
                serviceUrl:  appName ? this.getServiceUrl(appName) : undefined,
                tokenSet: undefined, // tokenSet.expires_in is Seconds
                tokenSetExpireTime: 0, // Seconds
                callbackListenerPort: 3000,
                mixpanelToken: this.getMixpanelToken(configName),
                idp: undefined
            },
            accessPropertiesByDotNotation: true,
            clearInvalidConfig: true    // if config is invalid, delete
        });

        if(! this.config.get('idp')) {
            thoumWarn('IdP not configured');
            
            // ref: https://github.com/anseki/readline-sync
            const readlineSync = require('readline-sync');
            const idps = ['Google', 'Microsoft'];
            const idpIndex: number = readlineSync.keyInSelect(idps, 'Please select your IdP:');

            const idpCheck = /^(Google|Microsoft)$/;
            if(idpCheck.test(idps[idpIndex])) {
                this.config.set('idp', idps[idpIndex]);
                this.config.set('authUrl', this.getAuthUrl(this.idp()));
                this.config.set('clientId', this.getClientId(configName, this.idp()));
            } else {
                thoumError('Invalid idp, please rerun command to try again');
                process.exit(1);
            }
        }

        if(configName == 'dev' && ! this.config.get('serviceUrl')) {
            thoumError(`Config not initialized (or is invalid) for dev environment: Must set serviceUrl, authUrl, IdP and ClientId in: ${this.config.path}`);
            process.exit(1);
        }
    }

    public configPath(): string {
        return this.config.path;
    }

    public mixpanelToken(): string {
        return this.config.get('mixpanelToken');
    }

    public callbackListenerPort(): number {
        return this.config.get('callbackListenerPort');
    }

    public serviceUrl(): string {
        return this.config.get('serviceUrl');
    }

    public authUrl(): string {
        return this.config.get('authUrl');
    }

    public tokenSet(): TokenSetParameters {
        return this.config.get('tokenSet');
    }

    // private until we have a reason to expose to app
    private idp(): IdP {
        return <IdP> this.config.get('idp');
    }

    public clientId(): string {
        return this.config.get('clientId');
    }

    public getAuthHeader(): string {
        return `${this.tokenSet().token_type} ${this.tokenSet().access_token}`
    }

    public tokenSetExpireTime(): number
    {
        return this.config.get('tokenSetExpireTime');
    }

    public setTokenSet(tokenSet?: TokenSet, tokenSetExpireTime?: number) {
        // TokenSet implements TokenSetParameters, makes saving it like
        // this safe to do.
        if(tokenSet)
            this.config.set('tokenSet', tokenSet);

        if(tokenSetExpireTime)
            this.config.set('tokenSetExpireTime', tokenSetExpireTime);
    }

    public logout()
    {
        this.config.delete('tokenSet');
        this.config.delete('tokenSetExpireTime');
    }

    private getAppName(configName: string) {
        switch(configName)
        {
            case 'prod':
                return 'app';
            case 'stage':
                return 'app-stage-4329423';
            default:
                return undefined;
        }
    }

    private getServiceUrl(appName: string) {

        return `https://${appName}.clunk80.com/`;
    }

    private getAuthUrl(idp: IdP) {
        switch(idp)
        {
            case IdP.Google:
                return 'https://accounts.google.com';
            case IdP.Microsoft:
                return 'https://login.microsoftonline.com/common/v2.0';
            default:
                return undefined;
        }
    }

    private getClientId(configName: string, idp: IdP)
    {
        switch(idp)
        {
            case IdP.Google:
                return '3249310342-9grnl0kjcv9f7ue7sq18t4l9l7jdu6ub.apps.googleusercontent.com';
            case IdP.Microsoft:
                switch(configName)
                {
                    case 'prod':
                        return '6744b3e0-7afe-45ee-8f62-64f4c29e3f07';
                    case 'stage':
                        return 'b9ad4f16-9967-477a-8491-9812ca4b68b7';
                    default:
                        return undefined;
                }
            default:
                return undefined;
        }
    }

    private getMixpanelToken(configName: string) {
        switch(configName)
        {
            case 'prod':
                return '3036a28cd1d9878a0f605bd1c76cdf96';
            default:
                return 'aef09ae2b274f4cccf33587a9c6552f0';
        }
    }
}
