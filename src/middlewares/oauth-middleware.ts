import { errors, UserinfoResponse } from "openid-client";
import { OAuthService } from "../oauth.service/oauth.service";
import { ConfigService } from "../config.service/config.service";
import { thoumError, thoumMessage, thoumWarn } from '../utils';

export async function oauthMiddleware(configService: ConfigService) : Promise<UserinfoResponse> {

    let ouath = new OAuthService(configService);

    let tokenSet = configService.tokenSet();

    // decide if we need to refresh or prompt user for login
    if(tokenSet)
    {
        if(configService.tokenSet().expired())
        {
            thoumMessage('Refreshing oauth');

            // refresh using existing creds
            await ouath.refresh()
            .then((newTokenSet) => configService.setTokenSet(newTokenSet))
            // Catch oauth related errors
            .catch((error: errors.OPError | errors.RPError) => {
                thoumError('Stale log in detected');
                thoumMessage('You need to log in, please run \'thoum login --help\'')
                configService.logout();
            });
        }
    } else {
        thoumWarn('You need to log in, please run \'thoum login --help\'');
        process.exit(0);
    }

    // Get user info from IdP
    let userInfo = await ouath.userInfo();
    return userInfo;
}