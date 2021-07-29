import { ConfigService } from '../config.service/config.service';
import { PolicyService,KubeService } from '../http.service/http.service';
import { Logger } from "../logger.service/logger";
import { Subject, SubjectType } from '../http.service/http.service.types';
import { v4 as uuidv4 } from 'uuid';
import { ClusterSummary, KubeClusterStatus } from "../types";
import { cleanExit } from './clean-exit.handler';
const { spawn } = require('child_process');


export async function removeUserHandler(userEmail: string, clusterName: string, clusterTargets: Promise<ClusterSummary[]>, configService: ConfigService, logger: Logger) {
    // First ensure we can lookup the user
    const kubeService = new KubeService(configService, logger);
    var userInfo = await kubeService.GetUserInfoFromEmail(userEmail);

    if (userInfo.email == 'unknown') {
        // Log an error
        logger.error(`Unable to find user with email: ${userEmail}`);
        await cleanExit(1, logger);
    }

    // Get the existing policy
    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();
    
    // Loop till we find the one we are looking for
    for (var policy of policies) {
        if (policy.name == clusterName) {
            // TODO: This can be done better then looping
            // Then remove the user from the policy
            var newSubjects = []
            for (var subject of policy.subjects) {
                if (subject.id != userInfo.id) {
                    newSubjects.push(subject)
                }
            }
            policy.subjects = newSubjects;

            // And finally update the policy
            await policyService.UpdateKubePolicy(policy)

            logger.info(`Removed ${userEmail} from ${clusterName} policy!`)
            await cleanExit(0, logger);
        }
    }
    
    // Log an error
    logger.error(`Unable to find the policy for cluster: ${clusterName}`);
    await cleanExit(1, logger);
}

