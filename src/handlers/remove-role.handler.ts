import { ConfigService } from '../config.service/config.service';
import { PolicyService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { ClusterSummary } from '../types';
import { cleanExit } from './clean-exit.handler';


export async function removeRoleHandler(clusterUserName: string, policyName: string, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
    // First get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();

    // Loop till we find the one we are looking for
    for (const policy of policies) {
<<<<<<< HEAD
        if (policy.name == policyName) {
=======
        if (policy.name == clusterName) {
>>>>>>> 9e71d7c (kubectl logs cancel (#130))
            // Now check if the role exists
            if (policy.context.clusterUsers[clusterUserName] === undefined) {
                logger.error(`No role ${clusterUserName} exist for policy: ${policyName}`);
                await cleanExit(1, logger);
            }
            // Then remove the role from the policy if it exists
<<<<<<< HEAD
            delete policy.context.clusterUsers[clusterUserName];
=======
            delete policy.context.clusterRoles[clusterRoleName];
>>>>>>> 9e71d7c (kubectl logs cancel (#130))

            // And finally update the policy
            await policyService.UpdateKubePolicy(policy);

<<<<<<< HEAD
            logger.info(`Removed ${clusterUserName} from ${policyName} policy!`);
=======
            logger.info(`Removed ${clusterRoleName} from ${clusterName} policy!`);
>>>>>>> 9e71d7c (kubectl logs cancel (#130))
            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy: ${policyName}`);
    await cleanExit(1, logger);
}

