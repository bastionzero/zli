import { Logger } from '../../services/logger/logger.service';
import { ConfigService } from '../../services/config/config.service';
import { cleanExit } from '../clean-exit.handler';
import { EnvironmentDetails } from '../../services/environment/environment.types';
import { PolicyQueryService } from '../../services/policy-query/policy-query.service';
import { ClusterDetails } from '../../services/kube/kube.types';
import { getTableOfDescribeCluster } from '../../utils';


export async function describeClusterHandler(
    clusterName: string,
    configService: ConfigService,
    logger: Logger,
    clusterTargets: Promise<ClusterDetails[]>,
    envs: Promise<EnvironmentDetails[]>
) {
    // First determine if the name passed is valid
    let clusterSummary: ClusterDetails = null;
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

    if (clusterPolicyInfo.policies.length === 0){
        logger.info('There are no available policies for this cluster.');
        await cleanExit(0, logger);
    }
    // regular table output
    const tableString = getTableOfDescribeCluster(clusterPolicyInfo.policies, clusterSummary.targetUsers, environment.name);
    console.log(tableString);
}