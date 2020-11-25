import { UserinfoResponse } from "openid-client";
import { OAuthService } from "../oauth.service/oauth.service";
import { ConfigService } from "../config.service/config.service";
import { thoumMessage } from "../cli-driver";

export async function oauthMiddleware(configService: ConfigService) : Promise<UserinfoResponse> {

    let ouath = new OAuthService(configService.authUrl(), configService.callbackListenerPort());

    // All times related to oauth are in epoch second
    const now: number = Math.floor(Date.now() / 1000);

    // decide if we need to refresh, login, or use existing token
    if(configService.tokenSet() && configService.tokenSet().expires_at < now && configService.tokenSetExpireTime() > now)
    {
        thoumMessage('Refreshing oauth');
        // refresh using existing creds
        let newTokenSet = await ouath.refresh(configService.tokenSet());
        configService.setTokenSet(newTokenSet);
    } else if(! configService.tokenSet() || configService.tokenSetExpireTime() < now) {
        thoumMessage('Log in required, opening browser');
        // renew with log in flow
        await ouath.login((tokenSet, expireTime) => configService.setTokenSet(tokenSet, expireTime));
    }

    // Get user info from IdP
    let userInfo = await ouath.userInfo(configService.tokenSet());
    return userInfo;
}