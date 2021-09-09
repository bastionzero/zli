import { TargetType, TargetUser } from '../common.types';
import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { GetTargetPolicyResponse, GetTargetPolicyRequest, KubeProxyResponse, KubeProxyRequest, GetAllPoliciesForClusterIdResponse, GetAllPoliciesForClusterIdRequest } from './policy-query.messages';
import { Verb } from './policy-query.types';

export class PolicyQueryService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/policy-query', logger);
    }

    public ListTargetOSUsers(targetId: string, targetType: TargetType, verb?: Verb, targetUser?: TargetUser): Promise<GetTargetPolicyResponse>
    {
        const request: GetTargetPolicyRequest = {
            targetId: targetId,
            targetType: targetType,
            verb: verb,
            targetUser: targetUser
        };

        return this.Post('target-connect', request);
    }

    public CheckKubeProxy(
        clusterName: string,
        clusterUser: string,
        environmentId: string,
    ): Promise<KubeProxyResponse>
    {
        const request: KubeProxyRequest = {
            clusterName: clusterName,
            clusterUser: clusterUser,
            environmentId: environmentId,
        };

        return this.FormPost('kube-tunnel', request);
    }

    public GetAllPoliciesForClusterId(
        clusterId: string,
    ): Promise<GetAllPoliciesForClusterIdResponse>
    {
        const request: GetAllPoliciesForClusterIdRequest = {
            clusterId: clusterId,
        };

        return this.FormPost('get-kube-policies', request);
    }
}