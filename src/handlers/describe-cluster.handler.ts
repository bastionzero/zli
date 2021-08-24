import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import { cleanExit } from './clean-exit.handler';
import * as types from '../types';
import { EnvironmentDetails } from '../http.service/http.service.types';
import { PolicyQueryService } from '../http.service/http.service';


export async function describeClusterHandler(
    clusterName: string,
    configService: ConfigService,
    logger: Logger,
    clusterTargets: Promise<types.ClusterSummary[]>,
    envs: Promise<EnvironmentDetails[]>
) {
    // First determine if the name passed is valid
    let clusterSummary: types.ClusterSummary = null;
    for (const cluster of await clusterTargets) {
        if (cluster.name == clusterName) {
            clusterSummary = cluster;
            break;
        }
    }

    if (clusterSummary == null) {
        logger.error(`Unable to find cluster with name: ${clusterName}`);
        await cleanExit(1, logger);
    }

    // Now match it up with the environment
    let environment: EnvironmentDetails = null;
    for (const env of await envs) {
        if (env.id == clusterSummary.environmentId) {
            environment = env;
            break;
        }
    }

    // Now make a query to see all policies associated with this cluster
    const policyService = new PolicyQueryService(configService, logger);
    const clusterPolicyInfo = await policyService.GetAllPoliciesForClusterId(clusterSummary.id);

    // Build our policies string
    let policiesString = '';
    for (const policy of clusterPolicyInfo.policies) {
        policiesString += policy.name + ',';
    }
    if (clusterPolicyInfo.policies.length != 0) {
        policiesString = policiesString.substring(0, policiesString.length - 1); // remove trailing ,
    }

    // Build our validUsers string
    let validUserString = '';
    for (const validUser of clusterSummary.targetUsers) {
        validUserString += validUser + ',';
    }
    validUserString = validUserString.substring(0, validUserString.length - 1); // remove trailing ,

    // Now we can print all the information we know
    logger.info(`Cluster information for: ${clusterName}`);
    logger.info(`    - Environment Name: ${environment.name}`);
    logger.info(`    - Policies using this cluster: ${policiesString}`);
    logger.info(`    - Valid Cluster Users: ${validUserString}`);
}