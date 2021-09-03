export interface GetTargetPolicyResponse
{
    allowed: boolean;
    allowedTargetUsers: TargetUser[];
    allowedVerbs: Verb[]
}

export interface GetTargetPolicyRequest
{
    targetId: string;
    targetType: TargetType;
    verb?: Verb;
    targetUser?: TargetUser;
}

export interface KubeProxyResponse {
    allowed: boolean;
}

export interface KubeProxyRequest {
    clusterName: string;
    clusterUser: string;
    environmentId: string;
}

export interface GetAllPoliciesForClusterIdResponse {
    policies: PolicySummary[]
}

export interface GetAllPoliciesForClusterIdRequest {
    clusterId: string;
}