import { ConfigService } from '../config.service/config.service';
import { PolicyService } from '../http.service/http.service';
import { Logger } from "../logger.service/logger";
import { KubernetesPolicyClusterRoles } from '../http.service/http.service.types';
import { v4 as uuidv4 } from 'uuid';
import { ClusterSummary, KubeClusterStatus } from "../types";
import { cleanExit } from './clean-exit.handler';
const { spawn } = require('child_process');


export async function addRoleHandler(clusterRoleName: string, clusterName: string, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
    // First get the existing policy
    const policyService = new PolicyService(configService, logger);
    var policies = await policyService.ListAllPolicies();
    
    // Loop till we find the one we are looking for
    for (var policy of policies) {
        if (policy.name == clusterName) {
            // Then add the role to the policy
            var clusterRoleToAdd: KubernetesPolicyClusterRoles = {
                name: clusterRoleName
            }
            policy.context.clusterRoles[clusterRoleName] = clusterRoleToAdd

            // And finally update the policy
            await policyService.UpdateKubePolicy(policy)

            logger.info(`Added ${clusterRoleName} to ${clusterName} policy!`)
            await cleanExit(0, logger);
        }
    }
    
    // Log an error
    logger.error(`Unable to find the policy for cluster: ${clusterName}`);
    await cleanExit(1, logger);
}

