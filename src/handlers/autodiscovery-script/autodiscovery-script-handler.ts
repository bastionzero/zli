import util from 'util';
import fs from 'fs';

import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { EnvironmentDetails } from '../../services/environment/environment.types';
import { AutoDiscoveryScriptService } from '../../services/auto-discovery-script/auto-discovery-script.service';
import yargs from 'yargs';
import { autoDiscoveryScriptArgs } from './autodiscovery-script.command-builder';
import { getEnvironmentFromName } from '../../../src/utils';


export async function autoDiscoveryScriptHandler(
    argv: yargs.Arguments<autoDiscoveryScriptArgs>,
    logger: Logger,
    configService: ConfigService,
    environments: Promise<EnvironmentDetails[]>
) {
    // Print deprecation warning
    logger.warn('Warning: zli autodiscovery-script is no longer supported and will be removed in a future release. Please use zli generate-bash instead.');

    const environmentNameArg = argv.environmentName;
    const outputFileArg = argv.outputFile;
    const agentVersionArg = argv.agentVersion;
    const envs = await environments;

    const environment = await getEnvironmentFromName(environmentNameArg, envs, logger);

    const autodiscoveryScriptService = new AutoDiscoveryScriptService(configService, logger);
    const autodiscoveryScriptResponse = await autodiscoveryScriptService.getAutodiscoveryScript(argv.operatingSystem, `TARGET_NAME=\"${argv.targetName}\"`, environment.id, agentVersionArg);

    if(outputFileArg) {
        await util.promisify(fs.writeFile)(outputFileArg, autodiscoveryScriptResponse.autodiscoveryScript);
    } else {
        logger.info(autodiscoveryScriptResponse.autodiscoveryScript);
    }
}