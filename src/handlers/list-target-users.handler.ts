import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import { PolicyService } from '../http.service/http.service';
import { KubernetesPolicyContext, PolicyType, TargetConnectContext } from '../http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';
import { getTableOfTargetUsers } from '../utils';

export async function listTargetUsersHandler(configService: ConfigService, logger: Logger, argv : any, policyName: string) {

    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();
    const targetUsers : string[] = [];
    const policy = policies.find(p => p.name == policyName);
    if (policy.type == PolicyType.KubernetesProxy) {
        const kubernetesPolicyContext = policy.context as KubernetesPolicyContext;
        Object.values(kubernetesPolicyContext.clusterUsers).forEach(
            clusterUser => targetUsers.push(clusterUser.name)
        );
    } else if (policy.type == PolicyType.TargetConnect) {
        const targetAccessContext = policy.context as TargetConnectContext;
        Object.values(targetAccessContext.targetUsers).forEach(
            targetUser => targetUsers.push(targetUser.userName));
    }

    if(!! argv.json) {
        // json output
        console.log(JSON.stringify(targetUsers));
    } else {
        if (targetUsers.length === 0){
            logger.info('There are no available target users');
            await cleanExit(0, logger);
        }
        // regular table output
        const tableString = getTableOfTargetUsers(targetUsers);
        console.log(tableString);
    }
}