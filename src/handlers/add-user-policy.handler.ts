import { ConfigService } from '../config.service/config.service';
import { PolicyService,KubeService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { PolicyType, Subject, SubjectType } from '../http.service/http.service.types';
import { ClusterSummary } from '../types';
import { cleanExit } from './clean-exit.handler';

// TODO : This currently supports only cluster users - this should be extended to target users
export async function addUserToPolicyHandler(userEmail: string, policyName: string, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
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

        if (policy.name == policyName) {

            if (policy.type !== PolicyType.KubernetesProxy){
                logger.error(`Adding user to policy ${policyName} failed. Support for adding users to ${policy.type} policies will be added soon.`);
                await cleanExit(1, logger);
            }
            // Then add the user to the policy
            const subjectToAdd: Subject = {
                id: userInfo.id,
                type: SubjectType.User
            };
            policy.subjects.push(subjectToAdd);

            // And finally update the policy
            await policyService.UpdateKubePolicy(policy);
            
            logger.info(`Added ${userEmail} to ${policyName} policy!`);

            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy for cluster: ${policyName}`);
    await cleanExit(1, logger);
}

