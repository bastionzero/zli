import { TargetUser } from '../../services/common.types';
import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { PolicyType, KubernetesPolicyClusterUsers, KubernetesPolicyContext, TargetConnectContext } from '../../services/policy/policy.types';
import { PolicyService } from '../../services/policy/policy.service';
import { cleanExit } from '../clean-exit.handler';

export async function addTargetUserHandler(targetUserName: string, policyName: string, configService: ConfigService, logger: Logger) {
    // First get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();

    // Loop till we find the one we are looking for
    const policy = policies.find(p => p.name == policyName);

    if (!policy) {
        // Log an error
        logger.error(`Unable to find policy with name: ${policyName}`);
        await cleanExit(1, logger);
    }

    switch (policy.type) {
    case PolicyType.KubernetesTunnel:
        // Then add the role to the policy
        const clusterUserToAdd: KubernetesPolicyClusterUsers = {
            name: targetUserName
        };
        const kubernetesPolicyContext = policy.context as KubernetesPolicyContext;

        // If this cluster role exists already
        if (kubernetesPolicyContext.clusterUsers[targetUserName] !== undefined) {
            logger.error(`Role ${targetUserName} exists already for policy: ${policyName}`);
            await cleanExit(1, logger);
        }
        kubernetesPolicyContext.clusterUsers[targetUserName] = clusterUserToAdd;

        // And finally update the policy
        policy.context = kubernetesPolicyContext;
        break;
    case PolicyType.TargetConnect:
        // Then add the role to the policy
        const targetUserToAdd: TargetUser = {
            userName: targetUserName
        };
        const targetConnectPolicyContext = policy.context as TargetConnectContext;
        const targetUsers = targetConnectPolicyContext.targetUsers as {[targetUser: string]: TargetUser};

        // If this target user exists already
        if (targetUsers[targetUserName] !== undefined) {
            logger.error(`Target user ${targetUserName} exists already for policy: ${policyName}`);
            await cleanExit(1, logger);
        }
        targetUsers[targetUserName] = targetUserToAdd;
        targetConnectPolicyContext.targetUsers = targetUsers;

        // And finally update the policy
        policy.context = targetConnectPolicyContext;
        break;
    default:
        logger.error(`Adding target user to policy ${policyName} failed. Adding target users to ${policy.type} policies is not currently supported.`);
        await cleanExit(1, logger);
        break;
    }

    await policyService.EditPolicy(policy);

    logger.info(`Added ${targetUserName} to ${policyName} policy!`);
    await cleanExit(0, logger);
}