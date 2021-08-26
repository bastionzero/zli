import { ConfigService } from '../config.service/config.service';
import { PolicyService, GroupsService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { GroupSummary, PolicyType } from '../http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';

export async function deleteGroupFromPolicyHandler(groupName: string, policyName: string, configService: ConfigService, logger: Logger) {
    // First ensure we can lookup the group
    const groupsService = new GroupsService(configService, logger);
    const groups = await groupsService.ListGroups();
    let groupSummary : GroupSummary = undefined;
    for (const group of groups){
        if (group.name == groupName)
            groupSummary = group;
    }
    if (groupSummary == undefined) {
        logger.error(`Unable to find group with name: ${groupName}`);
        await cleanExit(1, logger);
    }

    // Get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();

    // Loop till we find the one we are looking for
    for (const policy of policies) {
        if (policy.name == policyName) {
            if (policy.type !== PolicyType.KubernetesProxy && policy.type !== PolicyType.TargetConnect){
                logger.error(`Deleting group from policy ${policyName} failed. Support for deleting groups from ${policy.type} policies will be added soon.`);
                await cleanExit(1, logger);
            }

            // Then delete the group from the policy
            // TODO : Here index/splice can be used
            const newGroups = [];
            for (const group of policy.groups) {
                if (group.id != groupSummary.idPGroupId) {
                    newGroups.push(group);
                }
            }
            policy.groups = newGroups;

            // And finally update the policy
            await policyService.EditPolicy(policy);

            logger.info(`Deleted ${groupName} from ${policyName} policy!`);
            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy for cluster: ${policyName}`);
    await cleanExit(1, logger);
}

