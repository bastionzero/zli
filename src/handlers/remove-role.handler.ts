import { KubernetesPolicyContext, PolicyType } from '../http.service/http.service.types';
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
<<<<<<< HEAD
<<<<<<< HEAD
        if (policy.name == policyName) {
=======
        if (policy.name == clusterName) {
>>>>>>> 9e71d7c (kubectl logs cancel (#130))
=======
        if (policy.name == policyName) {
>>>>>>> 35eb4af (Refactor Kube Opa Policy and Commands (#137))
=======
        if (policy.name == policyName && policy.type == PolicyType.KubernetesProxy) {
>>>>>>> 06be4d5 (Added users listing functionality. Added policies listing functionality and policy type filter)
            // Now check if the role exists
            const kubernetesPolicyContext = policy.context as KubernetesPolicyContext;
            if (kubernetesPolicyContext.clusterUsers[clusterUserName] === undefined) {
                logger.error(`No role ${clusterUserName} exist for policy: ${policyName}`);
                await cleanExit(1, logger);
            }
            // Then remove the role from the policy if it exists
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
            delete policy.context.clusterUsers[clusterUserName];
=======
            delete policy.context.clusterRoles[clusterRoleName];
>>>>>>> 9e71d7c (kubectl logs cancel (#130))
=======
            delete policy.context.clusterUsers[clusterUserName];
>>>>>>> 724999e (Merged list-targets and list-clusters functionality. Fixed filtering for clusters. Added target users for list-targets)
=======
            delete kubernetesPolicyContext.clusterUsers[clusterUserName];
>>>>>>> 06be4d5 (Added users listing functionality. Added policies listing functionality and policy type filter)

            // And finally update the policy
            policy.context = kubernetesPolicyContext;
            await policyService.UpdateKubePolicy(policy);

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
            logger.info(`Removed ${clusterUserName} from ${policyName} policy!`);
=======
            logger.info(`Removed ${clusterRoleName} from ${clusterName} policy!`);
>>>>>>> 9e71d7c (kubectl logs cancel (#130))
=======
            logger.info(`Removed ${clusterUserName} from ${policyName} policy!`)
>>>>>>> 35eb4af (Refactor Kube Opa Policy and Commands (#137))
=======
            logger.info(`Removed ${clusterUserName} from ${policyName} policy!`);
>>>>>>> 724999e (Merged list-targets and list-clusters functionality. Fixed filtering for clusters. Added target users for list-targets)
            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy: ${policyName}`);
    await cleanExit(1, logger);
}

