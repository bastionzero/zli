import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import { cleanExit } from './clean-exit.handler';
import * as types from '../../src/types';
import { EnvironmentDetails } from '../../src/http.service/http.service.types';
import { PolicyQueryService } from '../../src/http.service/http.service';


export async function describeHandler(
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
    // Now make a query to see all the policy information
    const policyService = new PolicyQueryService(configService, logger);
    const clusterPolicyInfo = await policyService.DescribeKubeProxy(clusterName);

    // Build our clusterrole string
    let clusterRoleString = '';
    for (const clusterRole of clusterPolicyInfo.clusterRoles) {
        clusterRoleString += clusterRole.name + ',';
    }
    if (clusterPolicyInfo.clusterRoles.length != 0) {
        clusterRoleString = clusterRoleString.substring(0, clusterRoleString.length - 1); // remove trailing ,
    }

    // Build our validroles string
    let validRoleString = '';
    for (const validRole of clusterSummary.validRoles) {
        validRoleString += validRole + ',';
    }
    validRoleString = validRoleString.substring(0, validRoleString.length - 1); // remove trailing ,

    // Now we can print all the information we know
    logger.info(`Cluster information for: ${clusterName}`);
    logger.info(`    - Environment Name: ${environment.name}`);
    logger.info(`    - Cluster Roles Attached To Policy: ${clusterRoleString}`);
    logger.info(`    - Valid Cluster Roles: ${validRoleString}`);
}