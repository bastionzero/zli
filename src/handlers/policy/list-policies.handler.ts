import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';
import { getTableOfPolicies, parsePolicyType } from '../../utils';
import _ from 'lodash';
import { ApiKeyService } from '../../services/api-key/api-key.service';
import { ApiKeyDetails } from '../../services/api-key/api-key.types';
import { TargetSummary } from '../../services/common.types';
import { EnvironmentDetails } from '../../services/environment/environment.types';
import { GroupsService } from '../../services/groups/groups.service';
import { GroupSummary } from '../../services/groups/groups.types';
import { ClusterDetails } from '../../services/kube/kube.types';
import { PolicyService } from '../../services/policy/policy.service';
import { UserService } from '../../services/user/user.service';
import { UserSummary } from '../../services/user/user.types';
import yargs from 'yargs';
import { policyArgs } from './policy.command-builder';

export async function listPoliciesHandler(
    argv: yargs.Arguments<policyArgs>,
    configService: ConfigService,
    logger: Logger,
    ssmTargets: Promise<TargetSummary[]>,
    dynamicAccessConfigs: Promise<TargetSummary[]>,
    clusterTargets: Promise<ClusterDetails[]>,
    environments: Promise<EnvironmentDetails[]>
){
    const policyService = new PolicyService(configService, logger);
    const userService = new UserService(configService, logger);
    const apiKeyService = new ApiKeyService(configService, logger);
    const groupsService = new GroupsService(configService, logger);

    let policies = await policyService.ListAllPolicies();

    // If provided type filter, apply it
    if(!! argv.type) {
        const policyType = parsePolicyType(argv.type);
        policies = _.filter(policies,p => p.type == policyType);
    }

    // Fetch all the users, apiKeys, environments and targets
    // We will use that info to print the policies in a readable way
    const users = await userService.ListUsers();
    const userMap : { [id: string]: UserSummary } = {};
    users.forEach(userSummary => {
        userMap[userSummary.id] = userSummary;
    });

    const apiKeys = await apiKeyService.ListAllApiKeys();
    const apiKeyMap : { [id: string]: ApiKeyDetails } = {};
    apiKeys.forEach(apiKeyDetails => {
        apiKeyMap[apiKeyDetails.id] = apiKeyDetails;
    });

    const groupMap : { [id: string]: GroupSummary } = {};
    const groups = await groupsService.ListGroups();
    if (!!groups)
        groups.forEach(groupSummary => {
            groupMap[groupSummary.idPGroupId] = groupSummary;
        });

    const environmentMap : { [id: string]: EnvironmentDetails } = {};
    (await environments).forEach(environmentDetails => {
        environmentMap[environmentDetails.id] = environmentDetails;
    });

    const targetNameMap : { [id: string]: string } = {};
    (await ssmTargets).forEach(ssmTarget => {
        targetNameMap[ssmTarget.id] = ssmTarget.name;
    });
    (await dynamicAccessConfigs).forEach(dacs => {
        targetNameMap[dacs.id] = dacs.name;
    });
    (await clusterTargets).forEach(clusterTarget => {
        targetNameMap[clusterTarget.id] = clusterTarget.name;
    });

    if(!! argv.json) {
        // json output
        console.log(JSON.stringify(policies));
    } else {
        if (policies.length === 0){
            logger.info('There are no available policies');
            await cleanExit(0, logger);
        }
        // regular table output
        const tableString = getTableOfPolicies(policies, userMap, apiKeyMap, environmentMap, targetNameMap, groupMap);
        console.log(tableString);
    }

    await cleanExit(0, logger);
}