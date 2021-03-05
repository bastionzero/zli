import { errors } from 'openid-client';
import { OAuthService } from '../oauth.service/oauth.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../../src/logger.service/logger';

export async function oauthMiddleware(configService: ConfigService, logger: Logger) : Promise<void> {

    let ouath = new OAuthService(configService, logger);

    let tokenSet = configService.tokenSet();

    // decide if we need to refresh or prompt user for login
    if(tokenSet)
    {
        // if(configService.tokenSet().expired())
        if (true)
        {
            logger.debug('Refreshing oauth');

            // refresh using existing creds
            await ouath.refresh()
            .then((newTokenSet) => configService.setTokenSet(newTokenSet))
            // Catch oauth related errors
            .catch((error: errors.OPError | errors.RPError) => {
                logger.error('Stale log in detected');
                logger.info('You need to log in, please run \'zli login --help\'')
                // TODO trade of exception
                configService.logout();
                process.exit(1);
            })
            .catch((error: any) => {
                logger.error('Unexpected error during oauth refresh');
                logger.info('Please log in again');
                configService.logout();
                process.exit(1);
            });
        }
    } else {
        logger.warn('You need to log in, please run \'zli login --help\'');
        process.exit(1);
    }
}