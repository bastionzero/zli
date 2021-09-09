import util from 'util';
import fs from 'fs';

import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';
import { EnvironmentDetails } from '../../services/environment/environment.types';
import { AutoDiscoveryScriptService } from '../../services/auto-discovery-script/auto-discovery-script.service';
import yargs from 'yargs';
import { autoDiscoveryScriptArgs } from './autodiscovery-script.command-builder';


export async function autoDiscoveryScriptHandler(
    argv: yargs.Arguments<autoDiscoveryScriptArgs>,
    logger: Logger,
    configService: ConfigService,
    environments: Promise<EnvironmentDetails[]>
) {
    const environmentNameArg = argv.environmentName;
    const outputFileArg = argv.outputFile;
    const agentVersionArg = argv.agentVersion;
    const envs = await environments;

    const environment = envs.find(envDetails => envDetails.name == environmentNameArg);
    if (!environment) {
        logger.error(`Environment ${environmentNameArg} does not exist`);
        await cleanExit(1, logger);
    }

    const autodiscoveryScriptService = new AutoDiscoveryScriptService(configService, logger);
    const autodiscoveryScriptResponse = await autodiscoveryScriptService.getAutodiscoveryScript(argv.operatingSystem, argv.targetName, environment.id, agentVersionArg);

    if(outputFileArg) {
        await util.promisify(fs.writeFile)(outputFileArg, autodiscoveryScriptResponse.autodiscoveryScript);
    } else {
        logger.info(autodiscoveryScriptResponse.autodiscoveryScript);
    }
}