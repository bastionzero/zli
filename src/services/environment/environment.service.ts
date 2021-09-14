import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { CreateEnvironmentRequest, CreateEnvironmentResponse } from './environment.messages';
import { EnvironmentDetails } from './environment.types';

export class EnvironmentService extends HttpService {
    constructor(configService: ConfigService, logger: Logger) {
        super(configService, 'api/v1/environment/', logger);
    }

    public ListEnvironments(): Promise<EnvironmentDetails[]> {
        return this.Post('list', {});
    }

    public CreateEnvironment(req: CreateEnvironmentRequest): Promise<CreateEnvironmentResponse> {
        return this.Post<CreateEnvironmentRequest, CreateEnvironmentResponse>('create', req);
    }
}