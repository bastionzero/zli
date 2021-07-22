import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import { KubeService } from '../http.service/http.service';
import { cleanExit } from './clean-exit.handler';


const pem = require('pem')
const path = require('path');
const fs = require('fs');


export async function generateKubeYamlHandler(
    argv: any, 
    configService: ConfigService,
    logger: Logger
) {
    if (argv.clusterName == null) {
      logger.error('Please make sure you have passed a -clusterName before trying to generate a yaml!')
      await cleanExit(1, logger);
    }
  
    // Make our API client
    const kubeService = new KubeService(configService, logger);

    // Get our kubeYaml
    var kubeYaml = await kubeService.getKubeUnregisteredAgentYaml(argv.clusterName);

    // Show it to the user
    logger.info(kubeYaml.yaml);
}