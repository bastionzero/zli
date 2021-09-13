import util from 'util';
import fs from 'fs';
import { Logger } from '../../services/logger/logger.service';
import { ConfigService } from '../../services/config/config.service';
import { EnvironmentDetails } from '../../services/environment/environment.types';
import { AutoDiscoveryScriptService } from '../../services/auto-discovery-script/auto-discovery-script.service';
import { cleanExit } from '../clean-exit.handler';
import yargs from 'yargs';
import { generateBashArgs } from './generate-bash.command-builder';

export async function generateBashHandler(
    argv: yargs.Arguments<generateBashArgs>,
    logger: Logger,
    configService: ConfigService,
    environments: Promise<EnvironmentDetails[]>
) {
    let targetNameScript: string = '';
    if (argv.targetName === undefined) {
        switch (argv.targetNameScheme) {
        case 'do':
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
        default:
            // Compile-time exhaustive check
            // See: https://www.typescriptlang.org/docs/handbook/2/narrowing.html#exhaustiveness-checking
            const _exhaustiveCheck: never = argv.targetNameScheme;
            return _exhaustiveCheck;
        }
    } else {
        // Target name scheme option: Manual
        targetNameScript = `TARGET_NAME=\"${argv.targetName}\"`;
    }

    // Ensure that environment name argument is valid
    const envs = await environments;
    const environment = envs.find(envDetails => envDetails.name == argv.environment);
    if (!environment) {
        logger.error(`Environment ${argv.environment} does not exist`);
        await cleanExit(1, logger);
    }

    const autodiscoveryScriptService = new AutoDiscoveryScriptService(configService, logger);
    const autodiscoveryScriptResponse = await autodiscoveryScriptService.getAutodiscoveryScript(argv.os, targetNameScript, environment.id, argv.agentVersion);

    if (argv.outputFile) {
        await util.promisify(fs.writeFile)(argv.outputFile, autodiscoveryScriptResponse.autodiscoveryScript);
    } else {
        console.log(autodiscoveryScriptResponse.autodiscoveryScript);
    }
}