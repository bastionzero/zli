import { KubernetesPolicyContext, PolicyType, TargetConnectContext, TargetUser } from '../http.service/http.service.types';
import { ConfigService } from '../config.service/config.service';
import { PolicyService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';

export async function deleteTargetUserHandler(targetUserName: string, policyName: string, configService: ConfigService, logger: Logger) {
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
        // Now check if the role exists
        const kubernetesPolicyContext = policy.context as KubernetesPolicyContext;
        if (kubernetesPolicyContext.clusterUsers[targetUserName] === undefined) {
            logger.error(`No role ${targetUserName} exists for policy: ${policyName}`);
            await cleanExit(1, logger);
        }
        // Then remove the role from the policy if it exists
        delete kubernetesPolicyContext.clusterUsers[targetUserName];

        // And finally update the policy
        policy.context = kubernetesPolicyContext;
        break;
    case PolicyType.TargetConnect:
        const targetConnectContext = policy.context as TargetConnectContext;
        const targetUsers = targetConnectContext.targetUsers as {[targetUser: string]: TargetUser};
        if (targetUsers[targetUserName] === undefined) {
            logger.error(`No target user ${targetUserName} exists for policy: ${policyName}`);
            await cleanExit(1, logger);
        }

        // Then remove the role from the policy if it exists
        delete targetUsers[targetUserName];
        targetConnectContext.targetUsers = targetUsers;

        // And finally update the policy
        policy.context = targetConnectContext;
        break;
    default:
        logger.error(`Delete target user from policy ${policyName} failed. Deleting target users from ${policy.type} policies is not currently supported.`);
        await cleanExit(1, logger);
        break;
    }

    await policyService.EditPolicy(policy);

    logger.info(`Deleted ${targetUserName} from ${policyName} policy!`);
    await cleanExit(0, logger);
}

