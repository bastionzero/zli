import { ConfigService } from '../config.service/config.service';
import { PolicyService,KubeService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { Subject, SubjectType } from '../http.service/http.service.types';
import { ClusterSummary } from '../types';
import { cleanExit } from './clean-exit.handler';


export async function addUserHandler(userEmail: string, policyName: string, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
    // First ensure we can lookup the user
    const kubeService = new KubeService(configService, logger);

    let userInfo = null;
    try {
        userInfo = await kubeService.GetUserInfoFromEmail(userEmail);
    } catch (error) {
        logger.error(`Unable to find user with email: ${userEmail}`);
        await cleanExit(1, logger);

    }

    // Get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();

    // Loop till we find the one we are looking for
    for (const policy of policies) {
<<<<<<< HEAD
<<<<<<< HEAD
        if (policy.name == policyName) {
=======
        if (policy.name == clusterName) {
>>>>>>> 9e71d7c (kubectl logs cancel (#130))
=======
        if (policy.name == policyName) {
>>>>>>> 35eb4af (Refactor Kube Opa Policy and Commands (#137))
            // Then add the user to the policy
            const subjectToAdd: Subject = {
                id: userInfo.id,
                type: SubjectType.User
            };
            policy.subjects.push(subjectToAdd);

            // And finally update the policy
            await policyService.UpdateKubePolicy(policy);

<<<<<<< HEAD
<<<<<<< HEAD
            logger.info(`Added ${userEmail} to ${policyName} policy!`);
=======
            logger.info(`Added ${userEmail} to ${clusterName} policy!`);
>>>>>>> 9e71d7c (kubectl logs cancel (#130))
=======
            logger.info(`Added ${userEmail} to ${policyName} policy!`);
>>>>>>> 35eb4af (Refactor Kube Opa Policy and Commands (#137))
            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy for cluster: ${policyName}`);
    await cleanExit(1, logger);
}

