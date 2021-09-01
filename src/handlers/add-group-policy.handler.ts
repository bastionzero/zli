import { ConfigService } from '../config.service/config.service';
import { PolicyService, GroupsService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { Group, GroupSummary, PolicyType } from '../http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';

export async function addGroupToPolicyHandler(groupName: string, policyName: string, configService: ConfigService, logger: Logger) {
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
    const policy = policies.find(p => p.name == policyName);

    if (!policy) {
        // Log an error
        logger.error(`Unable to find policy with name: ${policyName}`);
        await cleanExit(1, logger);
    }

    if (policy.type !== PolicyType.KubernetesProxy && policy.type !== PolicyType.TargetConnect){
        logger.error(`Adding group to policy ${policyName} failed. Adding groups to ${policy.type} policies is not currently supported.`);
        await cleanExit(1, logger);
    }

    // If this group exists already
    const group = groups.find(g => g.name == groupSummary.name);
    if (group) {
        logger.error(`Group ${groupSummary.name} exists already for policy: ${policyName}`);
        await cleanExit(1, logger);
    }

    // Then add the group to the policy
    const groupToAdd: Group = {
        id: groupSummary.idPGroupId,
        name: groupSummary.name
    };
    policy.groups.push(groupToAdd);

    // And finally update the policy
    await policyService.EditPolicy(policy);

    logger.info(`Added ${groupName} to ${policyName} policy!`);
    await cleanExit(0, logger);
}

