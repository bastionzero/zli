import { OAuthService } from '../services/oauth/oauth.service';
import { ConfigService } from '../services/config/config.service';
import { Logger } from '../services/logger/logger.service';

export async function oauthMiddleware(configService: ConfigService, logger: Logger) : Promise<void> {

    const oauth = new OAuthService(configService, logger);

    await oauth.getIdToken();
}