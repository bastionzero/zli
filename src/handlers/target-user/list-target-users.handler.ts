import { Logger } from '../../services/logger/logger.service';
import { ConfigService } from '../../services/config/config.service';
import { cleanExit } from '../clean-exit.handler';
import { getTableOfTargetUsers } from '../../utils';
import { PolicyService } from '../../services/policy/policy.service';
import { PolicyType, KubernetesPolicyContext, TargetConnectContext } from '../../services/policy/policy.types';
import yargs from 'yargs';
import { targetUserArgs } from './target-user.command-builder';

export async function listTargetUsersHandler(configService: ConfigService, logger: Logger, argv : yargs.Arguments<targetUserArgs>, policyName: string) {

    const policyService = new PolicyService(configService, logger);
    const policies = await policyService.ListAllPolicies();
    const targetUsers : string[] = [];
    const policy = policies.find(p => p.name == policyName);
    if (policy.type == PolicyType.KubernetesTunnel) {
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