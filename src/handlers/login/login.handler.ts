import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { OAuthService } from '../../services/oauth/oauth.service';
import { cleanExit } from '../clean-exit.handler';
import { KeySplittingService } from '../../../webshell-common-ts/keysplitting.service/keysplitting.service';

import qrcode from 'qrcode';
import { MfaService } from '../../services/mfa/mfa.service';
import { MfaActionRequired } from '../../services/mfa/mfa.types';
import { UserService } from '../../services/user/user.service';
import yargs from 'yargs';
import { loginArgs } from './login.command-builder';

function requestMfaCode(): Promise<string> {
    const readline = require('readline');
    const rl = readline.createInterface({
        input: process.stdin,
        output: process.stdout,
    });

    return new Promise(resolve => rl.question('Enter MFA code from authenticator app and type Enter: ',
        (answer: string) => {
            rl.close();
            resolve(answer);
        }));
}

export async function loginHandler(configService: ConfigService, logger: Logger, argv: yargs.Arguments<loginArgs>, keySplittingService: KeySplittingService) {
    // Clear previous log in info
    configService.logout();
    await keySplittingService.generateKeysplittingLoginData();

    // Can only create oauth service after loginSetup completes
    const oAuthService = new OAuthService(configService, logger);
    if(! oAuthService.isAuthenticated())
    {
        logger.info('Login required, opening browser');

        // Create our Nonce
        const nonce = keySplittingService.createNonce();

        // Pass it in as we login
        await oAuthService.login((t) => {
            configService.setTokenSet(t);
            keySplittingService.setInitialIdToken(configService.getAuth());
        }, nonce);
    }

    // Register user log in and get User Session Id
    const userService = new UserService(configService, logger);
    const registerResponse = await userService.Register();
    configService.setSessionId(registerResponse.userSessionId);

    // Check if we must MFA and act upon it
    const mfaService = new MfaService(configService, logger);
    switch(registerResponse.mfaActionRequired)
    {
    case MfaActionRequired.NONE:
        break;
    case MfaActionRequired.TOTP:
        if(! argv.mfa)
        {
            logger.warn('MFA token required for this account');
            logger.info('Please try logging in again with \'--mfa <token>\' flag');
            configService.logout();
            await cleanExit(1, logger);
        }

        await mfaService.SendTotp(argv.mfa);

        break;
    case MfaActionRequired.RESET:
        logger.info('MFA reset detected, requesting new MFA token');
        logger.info('Please scan the following QR code with your device (Google Authenticator recommended) and enter code below.');

        const resp = await mfaService.ResetSecret(true);
        const data = await qrcode.toString(resp.mfaSecretUrl, {type: 'terminal', scale: 2});
        console.log(data);

        const token = await requestMfaCode();
        await mfaService.SendTotp(token);

        break;
    default:
        logger.warn(`Unexpected MFA response ${registerResponse.mfaActionRequired}`);
        break;
    }

    const me = await userService.Me();
    configService.setMe(me);

    logger.info(`Logged in as: ${me.email}, bzero-id:${me.id}, session-id:${registerResponse.userSessionId}`);

    await cleanExit(0, logger);
}