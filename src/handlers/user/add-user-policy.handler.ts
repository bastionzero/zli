import { KubeService } from '../../services/kube/kube.service';
import { PolicyService } from '../../services/policy/policy.service';
import { PolicyType, Subject, SubjectType } from '../../services/policy/policy.types';
import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';

export async function addUserToPolicyHandler(userEmail: string, policyName: string, configService: ConfigService, logger: Logger) {
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
            if (policy.type !== PolicyType.KubernetesTunnel && policy.type !== PolicyType.TargetConnect){
                logger.error(`Adding user to policy ${policyName} failed. Adding users to ${policy.type} policies is not currently supported.`);
                await cleanExit(1, logger);
            }
            // Then add the user to the policy
            const subjectToAdd: Subject = {
                id: userInfo.id,
                type: SubjectType.User
            };

            // If this group exists already
            for (const subject of policy.subjects) {
                if(subject.id == userInfo.id){
                    logger.error(`User ${userEmail} exists already for policy: ${policyName}`);
                    await cleanExit(1, logger);
                }
            }

            policy.subjects.push(subjectToAdd);

            // And finally update the policy
            await policyService.EditPolicy(policy);

            logger.info(`Added ${userEmail} to ${policyName} policy!`);

            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy for cluster: ${policyName}`);
    await cleanExit(1, logger);
}

