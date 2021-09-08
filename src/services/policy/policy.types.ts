export interface PolicySummary {
    id: string;
    name: string;
    metadata: PolicyMetadata
    type: PolicyType
    subjects: Subject[]
    groups: Group[]
    context: PolicyContext
}

export enum PolicyType {
    TargetConnect = 'TargetConnect',
    OrganizationControls = 'OrganizationControls',
    SessionRecording = 'SessionRecording',
    KubernetesTunnel = 'KubernetesTunnel'
}

export interface Subject {
    id: string;
    type: SubjectType;
}

export enum SubjectType {
    User = 'User',
    ApiKey = 'ApiKey'
}

export interface Group {
    id: string;
    name: string;
}

export type PolicyContext = TargetConnectContext | KubernetesPolicyContext;

export interface TargetConnectContext {
    targets: object;
    environments: object;
    targetUsers: object;
    verbs: object;
}

export interface KubernetesPolicyContext {
    clusterUsers: { [key: string]: KubernetesPolicyClusterUsers }
    environments: { [key: string]: PolicyEnvironment }
    clusters: { [key: string] : Cluster}
}

export interface Cluster {
    id: string
}

export interface KubernetesPolicyClusterUsers {
    name: string;
}


export interface PolicyMetadata {
    description: string;
}

export interface PolicyEnvironment {
    id: string;
}