import { ConfigService } from '../config.service/config.service';
import { PolicyService } from '../http.service/http.service';
import { Logger } from "../logger.service/logger";
import { KubernetesPolicyClusterUsers } from '../http.service/http.service.types';
import { v4 as uuidv4 } from 'uuid';
import { ClusterSummary, KubeClusterStatus } from "../types";
import { cleanExit } from './clean-exit.handler';


export async function removeRoleHandler(clusterUserName: string, clusterName: string, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
    // First get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();

    // Loop till we find the one we are looking for
    for (const policy of policies) {
        if (policy.name == clusterName) {
            // Now check if the role exists
            if (policy.context.clusterUsers[clusterUserName] === undefined) {
                logger.error(`No role ${clusterUserName} exist for policy for cluster: ${clusterName}`);
                await cleanExit(1, logger);
            }
            // Then remove the role from the policy if it exists
            delete policy.context.clusterUsers[clusterUserName]

            // And finally update the policy
            await policyService.UpdateKubePolicy(policy);

            logger.info(`Removed ${clusterUserName} from ${clusterName} policy!`)
            await cleanExit(0, logger);
        }
    }

    // Log an error
    logger.error(`Unable to find the policy for cluster: ${clusterName}`);
    await cleanExit(1, logger);
}

