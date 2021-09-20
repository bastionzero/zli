import util from 'util';
import fs from 'fs';
import { Logger } from '../../services/logger/logger.service';
import { ConfigService } from '../../services/config/config.service';
import { EnvironmentDetails } from '../../services/environment/environment.types';
import { getAutodiscoveryScript } from '../../services/auto-discovery-script/auto-discovery-script.service';
import { cleanExit } from '../clean-exit.handler';
import yargs from 'yargs';
import { generateBashArgs } from './generate-bash.command-builder';
import { TargetName } from '../../../webshell-common-ts/autodiscovery-script/autodiscovery-script.types';

export async function generateBashHandler(
    argv: yargs.Arguments<generateBashArgs>,
    logger: Logger,
    configService: ConfigService,
    environments: Promise<EnvironmentDetails[]>
) {
    let targetName: TargetName;

    if (argv.targetName === undefined) {
        switch (argv.targetNameScheme) {
        case 'do':
            targetName = { scheme: 'digitalocean' };
            break;
        case 'aws':
            targetName = { scheme: 'aws' };
            break;
        case 'time':
            targetName = { scheme: 'time' };
            break;
        case 'hostname':
            targetName = { scheme: 'hostname' };
            break;
        default:
            // Compile-time exhaustive check
            // See: https://www.typescriptlang.org/docs/handbook/2/narrowing.html#exhaustiveness-checking
            const _exhaustiveCheck: never = argv.targetNameScheme;
            return _exhaustiveCheck;
        }
    } else {
        targetName = { name: argv.targetName, scheme: 'manual' };
    }

    // Ensure that environment name argument is valid
    const envs = await environments;
    const environment = envs.find(envDetails => envDetails.name == argv.environment);
    if (!environment) {
        logger.error(`Environment ${argv.environment} does not exist`);
        await cleanExit(1, logger);
    }

    const script = await getAutodiscoveryScript(logger, configService, environment.id, targetName, argv.os, argv.agentVersion);

    if (argv.outputFile) {
        await util.promisify(fs.writeFile)(argv.outputFile, script);
    } else {
        console.log(script);
    }
}