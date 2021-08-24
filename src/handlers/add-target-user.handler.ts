import { ConfigService } from '../config.service/config.service';
import { PolicyService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { KubernetesPolicyClusterUsers, KubernetesPolicyContext, PolicyType } from '../http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';

// TODO : This currently supports only cluster users - this should be extended to target users
export async function addTargetUserHandler(clusterUserName: string, policyName: string, configService: ConfigService, logger: Logger) {
    // First get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();

    // Loop till we find the one we are looking for
    for (const policy of policies) {
        if (policy.name == policyName) {
            if (policy.type !== PolicyType.KubernetesProxy){
                logger.error(`Adding target user to policy ${policyName} failed. Support for adding target users to ${policy.type} policies will be added soon.`);
                await cleanExit(1, logger);
            }
            // Then add the role to the policy
            const clusterUserToAdd: KubernetesPolicyClusterUsers = {
                name: clusterUserName
            };
            const kubernetesPolicyContext = policy.context as KubernetesPolicyContext;
            kubernetesPolicyContext.clusterUsers[clusterUserName] = clusterUserToAdd;
            policy.context = kubernetesPolicyContext;

            // And finally update the policy
            policy.context = kubernetesPolicyContext;
            await policyService.UpdateKubePolicy(policy);

            logger.info(`Added ${clusterUserName} to ${policyName} policy!`);

            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy: ${policyName}`);
    await cleanExit(1, logger);
}

