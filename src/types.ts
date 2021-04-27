export enum TargetType {
    SSM = 'SSM',
    SSH = 'SSH',
    DYNAMIC = 'DYNAMIC'
}

export enum SessionState {
    Active = 'Active',
    Closed = 'Closed',
    Error = 'Error'
}

export enum IdP {
    Google = 'Google',
    Microsoft = 'Microsoft'
}

export enum SsmTargetStatus {
    NotActivated = 'NotActivated',
    Offline = 'Offline',
    Online = 'Online',
    Terminated = 'Terminated',
    Error = 'Error'
}

export interface TargetSummary
{
    id: string;
    name: string;
    environmentId: string;
    type: TargetType;
    agentVersion: string;
    status: SsmTargetStatus;
}

export interface ParsedTargetString
{
    type: TargetType;
    user: string;
    id: string;
    name: string;
    path: string;
    envId: string;
    envName: string;
}