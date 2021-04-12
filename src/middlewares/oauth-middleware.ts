import { errors } from 'openid-client';
import { OAuthService } from '../oauth.service/oauth.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../../src/logger.service/logger';
import { cleanExit } from '../../src/handlers/clean-exit.handler';

export async function oauthMiddleware(configService: ConfigService, logger: Logger) : Promise<void> {

    const ouath = new OAuthService(configService, logger);

    const tokenSet = configService.tokenSet();

    // decide if we need to refresh or prompt user for login
    if(tokenSet)
    {
        if(configService.tokenSet().expired())
        {
            try {
                logger.debug('Refreshing oauth tokens');

                const newTokenSet = await ouath.refresh();
                configService.setTokenSet(newTokenSet);
                logger.debug('Oauth tokens refreshed');
            } catch(e) {
                if(e instanceof errors.RPError || e instanceof errors.OPError) {
                    logger.error('Stale log in detected');
                    logger.info('You need to log in, please run \'zli login --help\'');
                    configService.logout();
                    cleanExit(1, logger);
                } else {
                    logger.error('Unexpected error during oauth refresh');
                    logger.info('Please log in again');
                    configService.logout();
                    cleanExit(1, logger);
                }
            }
        }
    } else {
        logger.warn('You need to log in, please run \'zli login --help\'');
        process.exit(1);
    }
}