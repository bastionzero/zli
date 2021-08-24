import { KubernetesPolicyContext, PolicyType } from '../http.service/http.service.types';
import { ConfigService } from '../config.service/config.service';
import { PolicyService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';

// TODO : This currently supports only cluster users - this should be extended to target users
export async function deleteTargetUserHandler(clusterUserName: string, policyName: string, configService: ConfigService, logger: Logger) {
    // First get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();

    // Loop till we find the one we are looking for
    for (const policy of policies) {
        if (policy.name == policyName) {
            if (policy.type !== PolicyType.KubernetesProxy){
                logger.error(`Deleting target user from policy ${policyName} failed. Support for deleting target users from ${policy.type} policies will be added soon.`);
                await cleanExit(1, logger);
            }
            // Now check if the role exists
            const kubernetesPolicyContext = policy.context as KubernetesPolicyContext;
            if (kubernetesPolicyContext.clusterUsers[clusterUserName] === undefined) {
                logger.error(`No role ${clusterUserName} exist for policy: ${policyName}`);
                await cleanExit(1, logger);
            }
            // Then remove the role from the policy if it exists
            delete kubernetesPolicyContext.clusterUsers[clusterUserName];


            // And finally update the policy
            policy.context = kubernetesPolicyContext;
            await policyService.UpdateKubePolicy(policy);

            logger.info(`Removed ${clusterUserName} from ${policyName} policy!`);

            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy: ${policyName}`);
    await cleanExit(1, logger);
}

