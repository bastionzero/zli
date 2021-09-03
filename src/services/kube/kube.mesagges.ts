export interface GetKubeUnregisteredAgentYamlResponse {
    yaml: string;
}

export interface GetKubeUnregisteredAgentYamlRequest {
    clusterName: string;
    labels: string;
    namespace: string;
    environmentId: string;
}

export interface GetUserInfoResponse {
    email: string;
    id: string;
}

export interface GetUserInfoRequest{
    email: string;
}