import { ConfigService } from '../config.service/config.service';
import { PolicyService, GroupsService } from '../http.service/http.service';
import { Logger } from '../logger.service/logger';
import { Group, GroupSummary, PolicyType } from '../http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';

// TODO : This currently supports only cluster groups - this should be extended to target groups
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
    for (const policy of policies) {
        if (policy.name == policyName) {
            if (policy.type !== PolicyType.KubernetesProxy){
                logger.error(`Adding group to policy ${policyName} failed. Support for adding groups to ${policy.type} policies will be added soon.`);
                await cleanExit(1, logger);
            }
            // Then add the group to the policy
            const groupToAdd: Group = {
                id: groupSummary.idPGroupId,
                name: groupSummary.name
            };
            policy.groups.push(groupToAdd);

            // And finally update the policy
            await policyService.UpdateKubePolicy(policy);

            logger.info(`Added ${groupName} to ${policyName} policy!`);
            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy for cluster: ${policyName}`);
    await cleanExit(1, logger);
}

