import { ApiKeyService, DynamicAccessConfigService, EnvironmentService, PolicyService, SsmTargetService, UserService } from '../http.service/http.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { cleanExit } from './clean-exit.handler';
import { getTableOfPolicies, parsePolicyType } from '../utils';
import { UserSummary, ApiKeyDetails, EnvironmentDetails } from '../http.service/http.service.types';
import _ from 'lodash';

export async function listPoliciesHandler(
    argv: any,
    configService: ConfigService,
    logger: Logger,
){
    const policyService = new PolicyService(configService, logger);
    const userService = new UserService(configService, logger);
    const apiKeyService = new ApiKeyService(configService, logger);
    const environmentService = new EnvironmentService(configService, logger);
    const ssmTargetService = new SsmTargetService(configService, logger);
    const dynamicAccessConfigService = new DynamicAccessConfigService(configService, logger);

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

    const environments = await environmentService.ListEnvironments();
    const environmentMap : { [id: string]: EnvironmentDetails } = {};
    environments.forEach(environmentDetails => {
        environmentMap[environmentDetails.id] = environmentDetails;
    });

    const ssmTargets = await ssmTargetService.ListSsmTargets(false);
    const dynamicAccessConfigs= await dynamicAccessConfigService.ListDynamicAccessConfigs();
    const targetNameMap : { [id: string]: string } = {};
    ssmTargets.forEach(ssmTarget => {
        targetNameMap[ssmTarget.id] = ssmTarget.name;
    });
    dynamicAccessConfigs.forEach(dacs => {
        targetNameMap[dacs.id] = dacs.name;
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
        const tableString = getTableOfPolicies(policies, userMap, apiKeyMap, environmentMap, targetNameMap);
        console.log(tableString);
    }

    await cleanExit(0, logger);
}