export interface ClusterSummary {
    id: string;
    clusterName: string;
    status: KubeClusterStatus;
    environmentId?: string;
    validUsers: string[];
    agentVersion: string;
    lastAgentUpdate: Date;
}

export enum KubeClusterStatus {
    NotActivated = 'NotActivated',
    Offline = 'Offline',
    Online = 'Online',
    Terminated = 'Terminated',
    Error = 'Error'
}

export interface ClusterDetails
{
    id: string;
    name: string;
    status: KubeClusterStatus;
    environmentId: string;
    targetUsers: string[];
    lastAgentUpdate: Date;
    agentVersion: string;
}