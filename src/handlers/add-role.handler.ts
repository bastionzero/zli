import { ConfigService } from '../config.service/config.service';
import { PolicyService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { KubernetesPolicyClusterUsers } from '../http.service/http.service.types';
import { ClusterSummary } from '../types';
import { cleanExit } from './clean-exit.handler';


export async function addRoleHandler(clusterUserName: string, policyName: string, force: boolean, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
    // First get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();

    // Loop till we find the one we are looking for
    for (const policy of policies) {
        if (policy.name == policyName) {
            // Then add the role to the policy
            const clusterUserToAdd: KubernetesPolicyClusterUsers = {
                name: clusterUserName
            };
            policy.context.clusterUsers[clusterUserName] = clusterUserToAdd;

            // And finally update the policy
            await policyService.UpdateKubePolicy(policy);

            logger.info(`Added ${clusterUserName} to ${policyName} policy!`);
            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy: ${policyName}`);
    await cleanExit(1, logger);
}

