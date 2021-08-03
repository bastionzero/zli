import { ConfigService } from '../config.service/config.service';
import { PolicyService } from '../http.service/http.service';
import { Logger } from "../logger.service/logger";
import { KubernetesPolicyClusterUsers } from '../http.service/http.service.types';
import { v4 as uuidv4 } from 'uuid';
import { ClusterSummary, KubeClusterStatus } from "../types";
import { cleanExit } from './clean-exit.handler';


export async function addRoleHandler(clusterUserName: string, clusterName: string, force: boolean, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
    // First get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();

    // Check if this is a valid cluster name
    var validUser = false;
    for (var clusterInfo of await clusterTargets) {
        if (clusterInfo.name == clusterName) {
            for (var possibleRole of clusterInfo.validUsers) {
                if (possibleRole == clusterUserName) {
                    validUser = true;
                }
            }
        }
    }

    // If this is not a valid role, and they have not passed the force flag, exit
    if (validUser == false && force != true) {
        logger.error(`The role chosen: ${clusterUserName} is not a valid role on the cluster ${clusterName}. If this is a mistake, please use the -f flag. Run zli describe <custerName> to see all valid cluster roles.`)
        await cleanExit(1, logger);
    }

    // Loop till we find the one we are looking for
    for (const policy of policies) {
        if (policy.name == clusterName) {
            // Then add the role to the policy
            var clusterUserToAdd: KubernetesPolicyClusterUsers = {
                name: clusterUserName
            }
            policy.context.clusterUsers[clusterUserName] = clusterUserToAdd
            
            // And finally update the policy
            await policyService.UpdateKubePolicy(policy);

            logger.info(`Added ${clusterUserName} to ${clusterName} policy!`)
            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy for cluster: ${clusterName}`);
    await cleanExit(1, logger);
}

