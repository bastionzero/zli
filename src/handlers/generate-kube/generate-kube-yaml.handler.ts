import { ConfigService } from '../../services/config/config.service';
import { EnvironmentDetails } from '../../services/environment/environment.types';
import { Logger } from '../../services/logger/logger.service';
import util from 'util';
import { cleanExit } from '../clean-exit.handler';
import { KubeService } from '../../services/kube/kube.service';
import yargs from 'yargs';
import { generateKubeArgs } from './generate-kube.command-builder';

const fs = require('fs');


export async function generateKubeYamlHandler(
    argv: yargs.Arguments<generateKubeArgs>,
    envs: Promise<EnvironmentDetails[]>,
    configService: ConfigService,
    logger: Logger
) {
    // First check all the required args
    if (argv.clusterName == null) {
        logger.error('Please make sure you have passed a -clusterName before trying to generate a yaml!');
        await cleanExit(1, logger);
    }

    const outputFileArg = argv.outputFile;

    // Make our API client
    const kubeService = new KubeService(configService, logger);

    // Format our labels if they exist
    const labels: { [index: string ]: string } = {};
    if (argv.labels != []) {
        for (const keyValueString of argv.labels) {
            const key = keyValueString.split(':')[0];
            const value = String(keyValueString.split(':')[1]);
            labels[key] = value;
        }
    }

    // If environment has been passed, ensure it's a valid envId
    if (argv.environmentId != null) {
        let validEnv = false;
        (await envs).forEach(env => {
            if (env.id == argv.environmentId) {
                validEnv = true;
            }
        });
        if (validEnv == false) {
            logger.error('The environment Id you passed is invalid.');
            await cleanExit(1, logger);
        }
    }

    // Get our kubeYaml
    const kubeYaml = await kubeService.getKubeUnregisteredAgentYaml(argv.clusterName, labels, argv.namespace, argv.environmentId);

    // Show it to the user or write to file
    if (outputFileArg) {
        await util.promisify(fs.writeFile)(outputFileArg, kubeYaml.yaml);
        logger.info(`Wrote yaml to output file: ${outputFileArg}`);
    } else {
        logger.info(kubeYaml.yaml);
    }
    await cleanExit(0, logger);
}