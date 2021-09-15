import { ConfigService } from '../config/config.service';
import { EnvironmentDetails } from '../environment/environment.types';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { GetAutodiscoveryScriptResponse, GetAutodiscoveryScriptRequest } from './auto-discovery-script.messages';
import { OperatingSystem, TargetName } from './auto-discovery-script.types';

export class AutoDiscoveryScriptService extends HttpService {
    constructor(configService: ConfigService, logger: Logger) {
        super(configService, 'api/v1/AutodiscoveryScript', logger);
    }

    public getAutodiscoveryScript(
        operatingSystem: string,
        targetNameScript: string,
        environmentId: string,
        agentVersion: string
    ): Promise<GetAutodiscoveryScriptResponse> {
        const request: GetAutodiscoveryScriptRequest = {
            apiUrl: `${this.configService.serviceUrl()}api/v1/`,
            targetNameScript: targetNameScript,
            envId: environmentId,
            agentVersion: agentVersion
        };

        return this.Post(`${operatingSystem}`, request);
    }
}

export async function getAutodiscoveryScript(
    logger: Logger,
    configService: ConfigService,
    environmentId: string,
    targetName: TargetName,
    operatingSystem: OperatingSystem,
    agentVersion: string
) {
    let targetNameScript: string = '';
    switch (targetName.scheme) {
    case 'digitalocean':
        targetNameScript = 'TARGET_NAME=$(curl http://169.254.169.254/metadata/v1/hostname)';
        break;
    case 'aws':
        targetNameScript = String.raw`
TOKEN=$(curl -s -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
TARGET_NAME=$(curl -s -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/instance-id)`;
        break;
    case 'time':
        targetNameScript = 'TARGET_NAME=target-$(date +"%m%d-%H%M%S")';
        break;
    case 'hostname':
        targetNameScript = 'TARGET_NAME=$(hostname)';
        break;
    case 'manual':
        targetNameScript = `TARGET_NAME=\"${targetName.name}\"`;
        break;
    default:
        // Compile-time exhaustive check
        // See: https://www.typescriptlang.org/docs/handbook/2/narrowing.html#exhaustiveness-checking
        const _exhaustiveCheck: never = targetName;
        return _exhaustiveCheck;
    }

    const autodiscoveryScriptService = new AutoDiscoveryScriptService(configService, logger);
    const autodiscoveryScriptResponse = await autodiscoveryScriptService.getAutodiscoveryScript(operatingSystem, targetNameScript, environmentId, agentVersion);

    return autodiscoveryScriptResponse.autodiscoveryScript;
}