import util from 'util';
import fs from 'fs';

import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';

import { EnvironmentDetails } from '../http.service/http.service.types';
import { AutoDiscoveryScriptService } from '../http.service/http.service';


export async function autoDiscoveryScriptHandler(
    argv: any,
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